/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  DEFAULT_CUSTOM_SIDEBAR_MENUS,
  normalizeCustomSidebarMenuUrl,
  parseCustomSidebarMenus,
  serializeCustomSidebarMenus,
} from '@/features/custom-sidebar-menu/lib/custom-sidebar-menus'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const customSidebarMenusSchema = z.object({
  menus: z.array(
    z.object({
      title: z.string().trim().min(1),
      url: z
        .string()
        .trim()
        .min(1)
        .refine((value) => Boolean(normalizeCustomSidebarMenuUrl(value)), {
          message: 'Invalid web URL',
        }),
    })
  ),
})

type CustomSidebarMenusFormValues = z.infer<typeof customSidebarMenusSchema>

type CustomSidebarMenusSectionProps = {
  value: string
}

export function CustomSidebarMenusSection(
  props: CustomSidebarMenusSectionProps
) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formDefaults = useMemo(
    () => ({ menus: parseCustomSidebarMenus(props.value) }),
    [props.value]
  )
  const initialSerialized = useMemo(
    () => serializeCustomSidebarMenus(formDefaults.menus),
    [formDefaults.menus]
  )
  const form = useForm<CustomSidebarMenusFormValues>({
    resolver: zodResolver(customSidebarMenusSchema),
    defaultValues: formDefaults,
  })
  const fields = useFieldArray({
    control: form.control,
    name: 'menus',
  })

  useEffect(() => {
    form.reset(formDefaults)
  }, [form, formDefaults])

  const onSubmit = async (values: CustomSidebarMenusFormValues) => {
    const serialized = serializeCustomSidebarMenus(values.menus)
    if (serialized === initialSerialized) return

    await updateOption.mutateAsync({
      key: 'CustomSidebarMenus',
      value: serialized,
    })
  }

  const resetToDefault = () => {
    form.reset({
      menus: DEFAULT_CUSTOM_SIDEBAR_MENUS.map((menu) => ({ ...menu })),
    })
  }

  return (
    <SettingsSection title={t('Custom sidebar menus')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            onReset={resetToDefault}
            isSaving={updateOption.isPending}
            resetLabel='Reset to default'
            saveLabel='Save custom menus'
          />

          <p
            data-settings-form-span='full'
            className='text-muted-foreground text-sm'
          >
            {t(
              'Configure links that appear in the sidebar and open inside the app.'
            )}
          </p>

          <div data-settings-form-span='full' className='space-y-3'>
            {fields.fields.length === 0 && (
              <div className='text-muted-foreground rounded-lg border border-dashed px-4 py-8 text-center text-sm'>
                {t('No custom menus configured.')}
              </div>
            )}

            {fields.fields.map((field, index) => (
              <div
                key={field.id}
                className='grid min-w-0 gap-3 rounded-lg border p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)_auto] md:items-start'
              >
                <FormField
                  control={form.control}
                  name={`menus.${index}.title`}
                  render={({ field: titleField }) => (
                    <FormItem>
                      <FormLabel>{t('Menu name')}</FormLabel>
                      <FormControl>
                        <Input
                          placeholder={t('For example: Image Generator')}
                          {...titleField}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name={`menus.${index}.url`}
                  render={({ field: urlField }) => (
                    <FormItem>
                      <FormLabel>{t('Link URL')}</FormLabel>
                      <FormControl>
                        <Input
                          placeholder='https://example.com/tool'
                          {...urlField}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Only HTTP and HTTPS links can be embedded.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button
                  type='button'
                  size='icon'
                  variant='destructive'
                  className='md:mt-6'
                  aria-label={t('Delete')}
                  title={t('Delete')}
                  onClick={() => fields.remove(index)}
                >
                  <Trash2 />
                </Button>
              </div>
            ))}

            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() => fields.append({ title: '', url: '' })}
            >
              <Plus />
              {t('Add menu')}
            </Button>
          </div>

          <p
            data-settings-form-span='full'
            className='text-muted-foreground text-xs'
          >
            {t(
              'Each item creates a sidebar entry and opens the link in an embedded page.'
            )}
          </p>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
