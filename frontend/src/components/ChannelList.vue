<template>
  <q-table
    :rows="dataList"
    :columns="columns"
    :rows-per-page-options="[20, 50, 100]"
    row-key="id"
    selection="multiple"
    v-model:selected="selected"
    :row-class="rowClass"
  >
    <template v-slot:top-right="props">
      <q-btn
        label="Select All"
        color="primary"
        @click="selectAll(props.rows)"
      />
    </template>
  </q-table>
</template>
    
<script setup lang="ts">
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
import { defineProps, ref, watch, computed, defineEmits } from 'vue';
const selected = ref<string[]>([]);
const emit = defineEmits<{
  (event: 'select', value: unknown): void;
  (event: 'selectAll', value: unknown[]): void;
}>();

interface Channel {
  // 保留自原始接口的字段
  description: string; // 假设这个字段是可选的，因为在提供的 JSON 中没有
  icon: string; // 同上
  id: string;
  joined_at?: string; // 同上，也可能是 Date 类型
  max_members?: number; // 同上
  member_count?: number; // 同上
  name: string;
  owner: boolean; // 同上
  owner_id: string;
  union_org_id: string; // 假设这个字段是可选的
  union_world_id: string; // 同上

  // 根据提供的 JSON 数据添加的字段
  application_id?: string; // 假设这个字段是可选的
  op_user_id?: string; // 同上
  parent_id: string; // 同上
  permissions: string; // 同上
  position: number; // 同上
  private_type: number; // 同上
  speak_permission: number; // 同上
  sub_type: number; // 同上
  type: number; // 同上
}

// 定义 props
const props = defineProps({
  dataList: {
    type: Array,
    default: () => [],
  },
});

// 观察 selected 引用的变化
watch(selected, (newValue) => {
  if (newValue.length === 1) {
    // 单项选择
    emit('select', newValue[0]);
  } else {
    // 多项选择或者没有选择
    emit('selectAll', newValue);
  }
});

// 计算属性，返回一个函数，该函数将根据行是否被选中来返回相应的类名
// 明确声明 props 参数的类型，确保类型安全
const rowClass = computed(() => {
  return (props: { row: Channel }) => {
    // 现在 TypeScript 知道 props.row 是 RowData 类型，props.row.id 是 string
    return selected.value.includes(props.row.id) ? 'highlighted' : '';
  };
});

const selectAll = (rows: Channel[]) => {
  if (selected.value.length === rows.length) {
    selected.value = []; // 取消全选
  } else {
    // 全选，但只选择每个Channel的id
    selected.value = rows.map((channel) => channel.id);
  }
  // 发出全选事件，传递当前选中项的id数组
  emit('selectAll', selected.value);
};

const columns = ref([
  // Name column
  {
    type: 'selection', // 这里指定列的类型为选择框
    align: 'center',
    sortable: false,
  },
  {
    name: 'name',
    required: true,
    label: '子频道名称',
    align: 'left',
    field: 'name',
    sortable: true,
  },
  {
    name: 'id',
    label: '子频道id',
    align: 'left',
    field: 'id',
    sortable: true,
  },
  {
    name: 'owner_id',
    label: '创建者id',
    align: 'right',
    field: 'owner_id',
    sortable: true,
  },
  {
    name: 'permissions',
    label: '权限等级',
    align: 'right',
    field: 'permissions',
    sortable: true,
  },
  {
    name: 'speak_permission',
    label: '发言权限',
    align: 'left',
    field: 'speak_permission',
    sortable: true,
  },
  {
    name: 'private_type',
    label: '可见性',
    align: 'center',
    field: 'private_type',
    sortable: true,
  },
]);
</script>
    
<style>
.table-icon {
  width: 30px; /* Adjust as needed */
  height: auto;
}
.highlighted {
  background-color: #569be9; /* 你选择的高亮颜色 */
}
</style>  