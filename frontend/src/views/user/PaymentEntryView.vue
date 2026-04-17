<template>
  <component :is="entryView" />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { useAppStore } from '@/stores'

const appStore = useAppStore()

const PaymentView = defineAsyncComponent(() => import('@/views/user/PaymentView.vue'))
const PurchaseSubscriptionView = defineAsyncComponent(
  () => import('@/views/user/PurchaseSubscriptionView.vue')
)

const entryView = computed(() =>
  appStore.cachedPublicSettings?.payment_enabled ? PaymentView : PurchaseSubscriptionView
)
</script>
