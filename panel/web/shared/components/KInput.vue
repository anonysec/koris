<template>
  <input
    :id="id"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :autocomplete="autocomplete"
    :aria-describedby="ariaDescribedby"
    :aria-disabled="disabled"
    class="k-input"
    :class="{ 'k-input--disabled': disabled }"
    @input="onInput"
  />
</template>

<script setup lang="ts">
export interface KInputProps {
  modelValue?: string | number
  type?: 'text' | 'number' | 'password' | 'email'
  placeholder?: string
  disabled?: boolean
  id?: string
  autocomplete?: string
  ariaDescribedby?: string
}

withDefaults(defineProps<KInputProps>(), {
  modelValue: '',
  type: 'text',
  placeholder: '',
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void
}>()

function onInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}
</script>

<style scoped>
.k-input {
  display: block;
  width: 100%;
  height: 36px;
  padding: 0 var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  font-family: var(--font-family);
  font-size: var(--text-base);
  line-height: var(--leading-normal);
  outline: none;
  transition:
    border-color var(--duration-normal) var(--ease-default),
    box-shadow var(--duration-normal) var(--ease-default);
}

.k-input::placeholder {
  color: var(--color-muted);
}

.k-input:focus-visible {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.25);
}

.k-input--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
