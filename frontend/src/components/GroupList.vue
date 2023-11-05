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

    <!-- Custom cell template for icon -->
    <template v-slot:body-cell-icon="props">
      <q-td :props="props">
        <img :src="props.row.icon" alt="Channel Icon" class="table-icon" />
      </q-td>
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
  channels: null; // 或者更精确的类型，如果它通常有一个值
  description: string;
  icon: string;
  id: string;
  joined_at: string; // 如果您需要操作日期，可能会考虑使用 Date 类型
  max_members: number;
  member_count: number;
  name: string;
  owner: boolean;
  owner_id: string;
  union_org_id: string;
  union_world_id: string;
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
  // Icon column
  {
    type: 'selection', // 这里指定列的类型为选择框
    align: 'center',
    sortable: false,
  },
  {
    name: 'icon',
    align: 'center',
    label: '图标',
    field: 'icon',
    sortable: false,
    type: 'selection',
  },
  // Name column
  {
    name: 'name',
    required: true,
    label: '频道名称',
    align: 'left',
    field: 'name',
    sortable: true,
  },
  // Member count column
  {
    name: 'member_count',
    label: '成员数',
    align: 'right',
    field: 'member_count',
    sortable: true,
  },
  // Joined at column
  {
    name: 'joined_at',
    label: '加入时间',
    align: 'left',
    field: 'joined_at',
    sortable: true,
  },
  // Max members column
  {
    name: 'max_members',
    label: '最大成员数',
    align: 'right',
    field: 'max_members',
    sortable: true,
  },
  // Owner column
  {
    name: 'owner',
    label: '所有者',
    align: 'center',
    field: 'owner',
    sortable: true,
  },
  // ID column
  {
    name: 'id',
    label: 'ID',
    align: 'left',
    field: 'id',
    sortable: true,
  },
  // Description column
  {
    name: 'description',
    label: '描述',
    align: 'left',
    field: 'description',
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
  background-color: #e0e0e0; /* 你选择的高亮颜色 */
}
</style>  