# 按分辨率按次计费设计

## 目标

在现有模型定价的 `Per-request` 计费方式中，新增“按媒体分辨率计费”的可视化配置和后端计费能力。

管理员通过 UI 表格配置价格，不直接填写 JSON。用户侧生成接口继续使用现有参数，不新增请求字段。

## 范围

本次包含：

- 图片模型：按图片分辨率和生成张数计费。
- 视频模型：按视频分辨率和秒数计费。
- 后台模型定价 UI：支持固定按次价格、图片分辨率价格、视频分辨率每秒价格。
- 后端规则存储、规则校验、额度计算、日志明细、异步任务计费快照。

本次不包含：

- 不给 `/v1/images/*` 或 `/v1/videos/*` 增加新的用户请求字段。
- 不在代码里写死 `gpt-image-2` 的默认价格。
- 不改变 token 计费和表达式计费语义。
- 不自动抓取上游官方价格表。

## 现有接口参数

用户生成接口不需要改。

图片计费读取现有字段：

- `model`
- `size`
- `n`

示例：

```json
{
  "model": "gpt-image-2",
  "prompt": "A city at night",
  "size": "2K",
  "n": 3
}
```

视频计费读取现有字段：

- `model`
- `size`
- `seconds`
- `duration`
- `metadata.resolution`

示例：

```json
{
  "model": "sora-2-pro",
  "prompt": "A train crossing a bridge",
  "size": "4K",
  "seconds": "10"
}
```

也支持现有 metadata 形式：

```json
{
  "model": "veo-example",
  "prompt": "A train crossing a bridge",
  "duration": 10,
  "metadata": {
    "resolution": "4k"
  }
}
```

## 计费模式

顶层模型计费模式保持不变：

- `Per-token`
- `Per-request`
- `Expression`

`Per-request` 增加三个子类型：

- `Fixed`：现有固定每次价格。
- `Image resolution`：分辨率到每张图片价格。
- `Video resolution`：分辨率到每秒价格。

同一个模型同时存在分辨率规则和旧的固定 `ModelPrice` 时，分辨率规则优先。没有分辨率规则时，继续走现有固定按次价格。

## 后台 UI

现有模型定价抽屉保留顶层标签：

```text
[ Per-token ] [ Per-request ] [ Expression ]
```

在 `Per-request` 内增加分段控件：

```text
[ Fixed ] [ Image resolution ] [ Video resolution ]
```

图片分辨率价格表：

```text
┌─────────┬─────────┬────────────────────┐
│ Enable  │ Level   │ Price per image    │
├─────────┼─────────┼────────────────────┤
│   ✓     │ 1K      │ $ 0.010            │
│   ✓     │ 2K      │ $ 0.020            │
│   ✓     │ 4K      │ $ 0.040            │
└─────────┴─────────┴────────────────────┘

Default resolution: [ 1K ▼ ]
Unknown resolution: Reject request
```

视频分辨率价格表：

```text
┌─────────┬─────────┬────────────────────┐
│ Enable  │ Level   │ Price per second   │
├─────────┼─────────┼────────────────────┤
│   ✓     │ 480     │ $ 0.020 / s        │
│   ✓     │ 980     │ $ 0.040 / s        │
│   ✓     │ 1K      │ $ 0.060 / s        │
│   ✓     │ 2K      │ $ 0.120 / s        │
│   ✓     │ 4K      │ $ 0.240 / s        │
└─────────┴─────────┴────────────────────┘

Default resolution: [ 480 ▼ ]
Unknown resolution: Reject request
```

保存规则：

- 只保存已启用且价格有效的行。
- 默认分辨率只能从已启用行中选择。
- 未知分辨率默认拒绝，不自动回退，避免误收费。
- 新规则在可视化 UI 中编辑；普通管理员不需要直接编辑底层结构化数据。

模型列表摘要：

- 固定价：`$0.01 / request`
- 图片：`Image · 1K $0.010/image · 2K $0.020/image · 4K $0.040/image`
- 视频：`Video · 480 $0.020/s · 980 $0.040/s · 1K $0.060/s`

## 后端配置

新增一个结构化配置，挂在现有 settings/config 系统下。UI 负责编辑该配置，正常可视化流程不暴露原始 JSON。

模型到规则的结构：

```go
type PerRequestPriceRule struct {
    MediaType         string             `json:"media_type"`         // image 或 video
    Unit              string             `json:"unit"`               // image 或 second
    Prices            map[string]float64 `json:"prices"`             // 标准分辨率 -> 美元单价
    DefaultResolution string             `json:"default_resolution"` // 必须存在于 Prices
    FallbackEnabled   bool               `json:"fallback_enabled"`   // 默认 false
}
```

第一版 UI 固定提供这些档位：

- 图片：`1K`、`2K`、`4K`
- 视频：`480`、`980`、`1K`、`2K`、`4K`

## 分辨率归一化

新增一个共享 resolver 负责标准化分辨率。图片和视频计费都调用它，channel adaptor 不重复实现这套判断。

图片别名：

- `1k`、`1K`、`1024x1024`、`1024x1536`、`1536x1024` -> `1K`
- `2k`、`2K`、`2048x2048` -> `2K`
- `4k`、`4K`、`4096x4096`、`3840x2160`、`2160x3840` -> `4K`

视频别名：

- `480`、`480p`、包含 `480` 的尺寸 -> `480`
- `980`、`980p`、包含 `980` 的尺寸 -> `980`
- `1k`、`1K`、`1080`、`1080p`、包含 `1080` 的尺寸 -> `1K`
- `2k`、`2K`、包含 `1440` 或 `2048` 的尺寸 -> `2K`
- `4k`、`4K`、`2160`、`2160p`、包含 `2160`、`3840` 或 `4096` 的尺寸 -> `4K`

没有传分辨率时，使用规则里的 `DefaultResolution`。

传了未知分辨率且 `FallbackEnabled=false` 时，在请求上游前返回本地 `400`。

## 计费计算

分辨率价格必须在预扣费前解析完成，并冻结为本次请求的计费快照。预扣费、结算、日志都使用同一个解析结果。

图片额度：

```text
quota = price_per_image * n * group_ratio * QuotaPerUnit
```

视频额度：

```text
quota = price_per_second * seconds * group_ratio * QuotaPerUnit
```

额度取整使用现有价格转 quota 的统一规则。

图片：

- `n` 在 resolver 中解析并计入快照。
- 缺少 `n` 或 `n=0` 时按现有默认值 `1`。
- 分辨率按次计费命中后，`image_handler.go` 不能再额外追加 `OtherRatios["n"]`，避免重复乘张数。

视频：

- 秒数在 resolver 中解析并计入快照。
- 优先读取 `seconds`，没有时读取 `duration`。
- 如果现有请求校验已经为特定模型设置默认秒数，则 resolver 使用该默认值；否则秒数缺失或非法时返回 `400`。
- 异步任务保存 resolved seconds，完成轮询时保持 `PerCallBilling=true`，不做 token 重算或 adaptor 差额结算。

## 计费快照

新增一个小型 resolved pricing snapshot，用于日志、任务存储和 retry 一致性。

```go
type ResolvedPerRequestPricing struct {
    Mode       string  `json:"mode"`       // resolution
    MediaType  string  `json:"media_type"` // image 或 video
    Unit       string  `json:"unit"`       // image 或 second
    Resolution string  `json:"resolution"`
    UnitPrice  float64 `json:"unit_price"`
    Quantity   float64 `json:"quantity"`  // 图片张数或视频秒数
    PriceUSD   float64 `json:"price_usd"` // UnitPrice * Quantity
    Quota      int     `json:"quota"`
}
```

同步图片请求：`PriceData` 携带 resolved snapshot，保证预扣费和响应后结算使用同一结果。

异步视频任务：`TaskBillingContext` 携带 resolved snapshot。任务失败退款、日志展示、remix 复用都读取这个快照。

## 错误处理

以下情况在请求上游前返回本地 `400`：

- 模型配置了分辨率规则，但请求媒体类型和规则不匹配。
- 分辨率无法归一化且不允许 fallback。
- 归一化后的分辨率没有配置价格。
- 图片张数非法。
- 视频秒数非法且没有现有校验默认值可用。
- 价格缺失、为负数或不是有限数字。

错误信息包含模型名、媒体类型、原始分辨率或标准分辨率，便于管理员排查配置。

## 日志展示

消费日志记录：

- billing mode：per-request resolution
- media type
- resolution
- unit price
- quantity：图片张数或视频秒数
- quota 转换前的美元价格
- group ratio
- final quota

任务日志通过 `TaskBillingContext` 记录同样的信息。

## 兼容性

没有配置分辨率规则时，现有行为不变。

现有固定按次 `ModelPrice` 继续可用。

Token 计费和表达式计费不变。

该功能使用现有 options/config 存储，不需要新增跨数据库迁移 SQL，因此兼容 SQLite、MySQL、PostgreSQL。

## 测试计划

后端单元测试：

- 图片 `1K`、`2K`、`4K` 命中配置价格。
- 图片别名正确归一化。
- 图片 `n` 只乘一次。
- 图片未知分辨率在请求上游前返回 `400`。
- 视频 `480`、`980`、`1K`、`2K`、`4K` 命中配置每秒价格。
- 视频别名正确归一化。
- 视频 `seconds` 和 `duration` 按优先级解析。
- 视频未知分辨率在请求上游前返回 `400`。
- 分组倍率只应用一次。
- 没有分辨率规则时，现有固定按次计费仍然工作。
- 分辨率规则优先于固定 `ModelPrice`。
- 任务计费上下文冻结 resolved price，并跳过完成阶段重算。

前端验证：

- `Per-request` 子类型切换保留当前子类型数据。
- 图片表只保存启用且有效的行。
- 视频表只保存启用且有效的行。
- 默认分辨率不能选择禁用行。
- 模型摘要正确显示固定、图片、视频计费。
- 可视化模式能保存结构化规则，管理员不需要手写 JSON。
