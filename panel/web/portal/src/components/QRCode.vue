<script setup lang="ts">
import { ref, watch, onMounted, nextTick } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import QRCodeLib from 'qrcode'

const props = defineProps<{
  value: string
  size?: number
  visible?: boolean
}>()

const { t } = useI18n()
const canvasRef = ref<HTMLCanvasElement | null>(null)
const qrSize = props.size || 240

async function drawQR() {
  if (!canvasRef.value || !props.value) return

  try {
    await QRCodeLib.toCanvas(canvasRef.value, props.value, {
      width: qrSize,
      margin: 2,
      color: {
        dark: '#000000',
        light: '#ffffff',
      },
    })
  } catch {
    // If QR generation fails (e.g. data too long), clear canvas
    const ctx = canvasRef.value.getContext('2d')
    if (ctx) {
      ctx.clearRect(0, 0, qrSize, qrSize)
    }
  }
}

function downloadPNG() {
  if (!canvasRef.value) return
  const link = document.createElement('a')
  link.download = 'xray-config-qr.png'
  link.href = canvasRef.value.toDataURL('image/png')
  link.click()
}

watch(() => [props.value, props.visible], () => {
  if (props.visible !== false) {
    nextTick(drawQR)
  }
}, { immediate: false })

onMounted(() => {
  if (props.visible !== false) {
    nextTick(drawQR)
  }
})
</script>
<template>
  <div v-if="visible !== false" class="qrcode">
    <canvas ref="canvasRef" class="qrcode__canvas" :width="qrSize" :height="qrSize" />
    <button class="qrcode__download" @click="downloadPNG" type="button">
      📥 {{ t('portal.xray.downloadQR') }}
    </button>
  </div>
</template>
<style scoped>
.qrcode {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  background: #ffffff;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}
.qrcode__canvas {
  border-radius: var(--radius-sm);
  image-rendering: pixelated;
  max-width: 100%;
  height: auto;
}
.qrcode__download {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-xs);
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s;
}
.qrcode__download:hover {
  background: var(--color-surface-2);
}
</style>
