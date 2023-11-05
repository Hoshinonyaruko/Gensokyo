<template>
  <div ref="dom" class="editor" />
</template>
<style scoped lang="scss">
@import '~@fontsource/roboto-mono/index.css';

.editor {
  font-family: 'Roboto Mono', monospace, serif;
}
</style>
<script setup lang="ts">
import { onMounted, ref, watch, nextTick } from 'vue';
import { editor } from 'monaco-editor/esm/vs/editor/editor.api.js';

const dom = ref<HTMLElement>(),
  props = defineProps<{
    modelValue: string;
    language: string;
    theme?: string;
  }>(),
  emit = defineEmits(['update:modelValue']);

let instance: editor.IStandaloneCodeEditor;
let preventUpdate = false; // 初始化preventUpdate

onMounted(() => {
  if (dom.value) {
    // 检查dom.value是否为null或undefined
    instance = editor.create(dom.value, {
      value: props.modelValue,
      language: props.language,
      theme: props.theme,
      fontFamily: 'Roboto Mono',
    });

    // 监听内容变化事件
    instance.onDidChangeModelContent(async () => {
      preventUpdate = true; // 设置标志以避免watch触发setValue
      const value = instance.getValue();
      emit('update:modelValue', value);
      await nextTick(); // 等待nextTick完成，处理返回的promise
      preventUpdate = false; // 重置preventUpdate标志
    });
  } else {
    // 如果dom.value是null，处理错误或提供一个备用方案
    console.error('Editor DOM element is not available.');
  }
});

// 监听props.modelValue变化
watch(
  () => props.modelValue,
  (newValue) => {
    if (!preventUpdate) {
      // 仅在更新不是由用户输入触发时，才执行setValue
      instance.setValue(newValue);
    }
  }
);

// 如果还有其他watchers或逻辑，它们应该和onMounted并列
</script>