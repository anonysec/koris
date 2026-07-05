<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'

const props = withDefaults(defineProps<{
  mode?: 'in-out' | 'out-in'
  duration?: number
  disabled?: boolean
}>(), {
  mode: 'out-in',
  duration: 200,
  disabled: false,
})

const transitionName = ref('fade')
const router = useRouter()

// Detect navigation direction based on route depth or meta.order
router.beforeEach((to, from) => {
  if (props.disabled) {
    transitionName.value = ''
    return
  }

  const toDepth = to.path.split('/').filter(Boolean).length
  const fromDepth = from.path.split('/').filter(Boolean).length
  const toOrder = (to.meta?.order as number) ?? toDepth
  const fromOrder = (from.meta?.order as number) ?? fromDepth

  if (toOrder > fromOrder) {
    transitionName.value = 'slide-left'
  } else if (toOrder < fromOrder) {
    transitionName.value = 'slide-right'
  } else {
    transitionName.value = 'fade'
  }
})
</script>

<template>
  <router-view v-slot="{ Component }">
    <Transition
      :name="disabled ? '' : transitionName"
      :mode="mode"
      :duration="duration"
    >
      <component :is="Component" />
    </Transition>
  </router-view>
</template>

<style scoped>
/* Fade */
.fade-enter-active,
.fade-leave-active {
  transition: opacity v-bind('duration + "ms"') ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Slide Left (forward navigation) */
.slide-left-enter-active,
.slide-left-leave-active {
  transition: all v-bind('duration + "ms"') ease;
}
.slide-left-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* Slide Right (back navigation) */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: all v-bind('duration + "ms"') ease;
}
.slide-right-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}
.slide-right-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

/* Respect reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  .fade-enter-active,
  .fade-leave-active,
  .slide-left-enter-active,
  .slide-left-leave-active,
  .slide-right-enter-active,
  .slide-right-leave-active {
    transition-duration: 0ms !important;
  }
}
</style>
