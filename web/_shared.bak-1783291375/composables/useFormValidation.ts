import { ref, computed, watch, nextTick, type Ref, type ComputedRef } from 'vue'
import type { ValidationRule } from '@koris/types/components'

export interface UseFormValidationOptions<T extends Record<string, any>> {
  initialValues: T
  rules: Partial<Record<keyof T, ValidationRule[]>>
  validateOnChange?: boolean
}

export interface UseFormValidationReturn<T extends Record<string, any>> {
  values: Ref<T>
  errors: Ref<Partial<Record<keyof T, string[]>>>
  touched: Ref<Partial<Record<keyof T, boolean>>>
  isValid: ComputedRef<boolean>
  isDirty: ComputedRef<boolean>
  validate(): boolean
  validateField(field: keyof T): boolean
  reset(): void
  setFieldValue(field: keyof T, value: any): void
  setFieldTouched(field: keyof T): void
}

export function useFormValidation<T extends Record<string, any>>(
  options: UseFormValidationOptions<T>
): UseFormValidationReturn<T> {
  const { initialValues, rules, validateOnChange = false } = options

  const values = ref<T>({ ...initialValues }) as Ref<T>
  const errors = ref<Partial<Record<keyof T, string[]>>>({}) as Ref<Partial<Record<keyof T, string[]>>>
  const touched = ref<Partial<Record<keyof T, boolean>>>({}) as Ref<Partial<Record<keyof T, boolean>>>

  function validateSingleField(field: keyof T): string[] {
    const fieldRules = rules[field]
    if (!fieldRules || fieldRules.length === 0) return []

    const fieldErrors: string[] = []
    const value = values.value[field]

    for (const rule of fieldRules) {
      let valid = true
      switch (rule.type) {
        case 'required':
          valid = value !== '' && value !== null && value !== undefined
          break
        case 'minLength':
          valid = typeof value === 'string' && value.length >= rule.value
          break
        case 'maxLength':
          valid = typeof value === 'string' && value.length <= rule.value
          break
        case 'pattern':
          valid = new RegExp(rule.value).test(String(value ?? ''))
          break
        case 'custom':
          valid = rule.validator ? rule.validator(value) : true
          break
      }
      if (!valid) fieldErrors.push(rule.message)
    }
    return fieldErrors
  }

  function validateField(field: keyof T): boolean {
    const fieldErrors = validateSingleField(field)
    errors.value[field] = fieldErrors.length > 0 ? fieldErrors : undefined
    return fieldErrors.length === 0
  }

  function validate(): boolean {
    let allValid = true
    const allErrors: Partial<Record<keyof T, string[]>> = {}

    for (const field of Object.keys(rules) as Array<keyof T>) {
      const fieldErrors = validateSingleField(field)
      if (fieldErrors.length > 0) {
        allErrors[field] = fieldErrors
        allValid = false
      }
    }

    errors.value = allErrors

    // Focus first invalid field on submission failure
    if (!allValid) {
      const firstInvalidField = Object.keys(allErrors)[0]
      if (firstInvalidField) {
        nextTick(() => {
          const el = document.getElementById(`field-${String(firstInvalidField)}`)
          el?.focus()
        })
      }

      // Announce errors via ARIA live region
      announceErrors(allErrors)
    }

    return allValid
  }

  function announceErrors(fieldErrors: Partial<Record<keyof T, string[]>>): void {
    const errorCount = Object.keys(fieldErrors).length
    if (errorCount === 0) return

    // Create or find existing ARIA live region
    let liveRegion = document.getElementById('form-validation-live-region')
    if (!liveRegion) {
      liveRegion = document.createElement('div')
      liveRegion.id = 'form-validation-live-region'
      liveRegion.setAttribute('role', 'alert')
      liveRegion.setAttribute('aria-live', 'assertive')
      liveRegion.setAttribute('aria-atomic', 'true')
      liveRegion.style.position = 'absolute'
      liveRegion.style.width = '1px'
      liveRegion.style.height = '1px'
      liveRegion.style.padding = '0'
      liveRegion.style.margin = '-1px'
      liveRegion.style.overflow = 'hidden'
      liveRegion.style.clip = 'rect(0, 0, 0, 0)'
      liveRegion.style.whiteSpace = 'nowrap'
      liveRegion.style.border = '0'
      document.body.appendChild(liveRegion)
    }

    const messages = Object.entries(fieldErrors)
      .filter(([, errs]) => errs && errs.length > 0)
      .map(([field, errs]) => `${String(field)}: ${(errs as string[]).join(', ')}`)
      .join('. ')

    liveRegion.textContent = `Form has ${errorCount} ${errorCount === 1 ? 'error' : 'errors'}. ${messages}`
  }

  function reset(): void {
    values.value = { ...initialValues } as any
    errors.value = {}
    touched.value = {}
  }

  function setFieldValue(field: keyof T, value: any): void {
    (values.value as any)[field] = value
  }

  function setFieldTouched(field: keyof T): void {
    touched.value[field] = true
  }

  const isValid = computed(() => {
    for (const field of Object.keys(rules) as Array<keyof T>) {
      const fieldErrors = validateSingleField(field)
      if (fieldErrors.length > 0) return false
    }
    return true
  })

  const isDirty = computed(() => {
    for (const key of Object.keys(initialValues) as Array<keyof T>) {
      if (values.value[key] !== initialValues[key]) return true
    }
    return false
  })

  // Watch for changes if validateOnChange is enabled
  if (validateOnChange) {
    watch(values, () => {
      for (const field of Object.keys(rules) as Array<keyof T>) {
        if (touched.value[field]) {
          validateField(field)
        }
      }
    }, { deep: true })
  }

  return {
    values,
    errors,
    touched,
    isValid,
    isDirty,
    validate,
    validateField,
    reset,
    setFieldValue,
    setFieldTouched,
  }
}
