<template>
  <div class="k-select-wrapper">
    <select
      :id="id"
      :value="modelValue"
      :disabled="disabled"
      :aria-describedby="ariaDescribedby"
      :aria-disabled="disabled"
      class="k-select"
      :class="{ 'k-select--disabled': disabled, 'k-select--placeholder': !modelValue }"
      @change="onChange"
    >
      <option v-if="placeholder" value="" disabled>
        {{ placeholder }}
      </option>
      <option
        v-for="option in options"
        :key="option.value"
        :value="option.value"
      >
        {{ option.label }}
      </option>
    </select>
    <span class="k-select__arrow" aria-hidden="true">
      <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
        <path
          d="M3 4.5L6 7.5L9 4.5"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </span>
  </div>
</template>

<script setup lang="ts">
export interface KSelectOption {
  label: string
  value: string | number
}

export interface KSelectProps {
  modelValue?: string | number
  options?: KSelectOption[]
  placeholder?: string
  disabled?: boolean
  id?: string
  ariaDescribedby?: string
}

withDefaults(defineProps<KSelectProps>(), {
  modelValue: '',
  options: () => [],
  placeholder: '',
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void
}>()

function onChange(event: Event) {
  const target = event.target as HTMLSelectElement
  emit('update:modelValue', target.value)
}
</script>

<style scoped>
.k-select-wrapper {
  position: relative;
  display: block;
  width: 100%;
}

.k-select {
  display: block;
  width: 100%;
  height: 36px;
  padding: 0 var(--space-8) 0 var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  font-family: var(--font-family);
  font-size: var(--text-base);
  line-height: var(--leading-normal);
  outline: none;
  appearance: none;
  -webkit-appearance: none;
  cursor: pointer;
  transition:
    border-color var(--duration-normal) var(--ease-default),
    box-shadow var(--duration-normal) var(--ease-default);
}

.k-select--placeholder {
  color: var(--color-muted);
}

.k-select:focus-visible {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.25);
}

.k-select--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.k-select__arrow {
  position: absolute;
  top: 50%;
  right: var(--space-3);
  transform: translateY(-50%);
  pointer-events: none;
  color: var(--color-muted);
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
