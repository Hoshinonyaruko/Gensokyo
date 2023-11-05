<template>
  <q-page class="flex flex-center">
    <div class="q-pa-md full-width">
      <q-select
        v-model="selectedType"
        :options="typeOptions"
        label="选择列表类型"
        outlined
      />
    </div>
    <q-banner v-if="loading" class="q-pa-md" inline-actions dense>
      <template v-slot:avatar>
        <q-spinner color="primary" />
      </template>
      加载中...
    </q-banner>
    <q-banner
      v-if="error"
      class="q-pa-md bg-negative text-white"
      inline-actions
      dense
    >
      {{ errorMessage }}
    </q-banner>
    <GroupList
      v-if="!loading && !error"
      :data-list="groupList"
      @select="handleSelectItem"
      @selectAll="handleSelectAll"
      @row-click="handleRowClick"
    ></GroupList>
    <div class="q-pa-md full-width row items-center">
      <q-input v-model="message" label="发送消息" outlined class="col-9" />
      <q-btn
        :disabled="!selectedItems.length || !message"
        label="发送"
        @click="sendMessage"
        color="primary"
        class="col-3"
      />
      <q-btn
        :disabled="currentPage <= 1"
        icon="chevron_left"
        @click="previousPage"
        label="上一页"
      />
      <q-btn
        :disabled="currentPage >= totalPages"
        icon-right="chevron_right"
        @click="nextPage"
        label="下一页"
      />
    </div>
    <div class="q-pa-md full-width">
      <q-pagination
        v-model="currentPage"
        :max="totalPages"
        :max-pages="11"
        class="justify-center"
      />
    </div>
  </q-page>
</template>
<script setup lang="ts">
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
import { ref, watch, reactive, computed, onMounted } from 'vue';
import { api } from 'src/boot/axios';
import GroupList from 'src/components/GroupList.vue';
import ChannelList from 'src/components/ChannelList.vue';
import { useRouter } from 'vue-router';

const $router = useRouter();
const props = defineProps<{ uin: number }>();
const selectedType = ref('群组');
const groupList = ref([]);
const channelList = ref([]);
const selectedItems = ref<string[]>([]);
const message = ref('');
const loading = ref(false);
const error = ref(false);

// Computed for error message to make it reactive
const errorMessage = computed(() => {
  return error.value ? `获取群组数据失败，请稍后再试。` : '';
});

// 分页状态和逻辑
const currentPage = ref(1);
const totalPages = ref(1000);
const pager = reactive({
  Before: '',
  After: '',
  Limit: '100', // 假设每页30条
});

// Fetch data based on type
async function fetchDataByType(type: string): Promise<void> {
  loading.value = true;
  error.value = false; // 重置错误状态
  try {
    const apiName = type === '群组' ? 'get_guild_list' : 'get_channel_list';
    const response = await api.accountApiApiUinApiPost(
      props.uin,
      apiName, // API 名称决定调用哪个接口
      { ...pager } // 这里直接使用 pager 作为请求体发送
    );

    // 从响应中解构 data 和 totalPages
    const { data } = response;
    groupList.value = (data as { data: any[] }).data;
    console.error(groupList.value);
    totalPages.value = 1000; // 假设后端会返回总页数
  } catch (e) {
    error.value = true; // 在这里只设置error的状态，errorMessage会根据它计算得出
    console.error(e); // 输出错误到控制台
  } finally {
    loading.value = false;
  }
}
interface Row {
  channels: null; //
  description: string;
  icon: string;
  id: string;
  joined_at: string; // 这里使用 string 类型，但您也可以转换为 Date 类型
  // ...其他属性，根据您的实际需要来定义
}

function handleRowClick(evt: MouseEvent, row: Row, index: number): void {
  // 使用row对象的属性
  console.log(row.description); // 这应该正常工作，并且现在类型是安全的

  $router
    .push({
      name: 'channellist',
      params: { uin: props.uin, channelid: row.id },
    })
    .catch((error) => {
      console.error(error);
    });
}

// 更新pager以获取下一页
const getNextPage = async (lastItemId: string) => {
  // 设置after为最后一个item的id，before清空
  pager.After = lastItemId;
  pager.Before = '';
  // 可以调用fetchDataByType来获取下一页的数据
  await fetchDataByType(selectedType.value);
};

// 更新pager以获取上一页
const getPreviousPage = async (firstItemId: string) => {
  // 设置before为第一个item的id，after清空
  pager.Before = firstItemId;
  pager.After = '';
  // 调用fetchDataByType来获取上一页的数据
  await fetchDataByType(selectedType.value);
};

// 下一页按钮的点击事件处理函数
const nextPage = async () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value++;
    const lastItemId =
      selectedType.value === '群组'
        ? String(groupList.value[groupList.value.length - 1].id)
        : String(channelList.value[channelList.value.length - 1].id);
    await getNextPage(lastItemId).catch((e) => console.error(e));
  }
};

// 当用户选择单个项目时调用
const handleSelectItem = (selectedItemId: string) => {
  // 更新响应式状态以反映当前选中的项目
  selectedItems.value = [selectedItemId];
  // 此外，可能还需要执行其他操作，比如发送请求或更新 UI 等
  // ...其他逻辑
};

// 当用户选择所有项目或取消选择所有项目时调用
const handleSelectAll = (selectedItemIds: string[]) => {
  // 更新响应式状态以反映当前选中的所有项目
  selectedItems.value = selectedItemIds;
  // 此外，可能还需要执行其他操作，比如批量处理或批量请求等
  // ...其他逻辑
};

// 上一页按钮的点击事件处理函数
const previousPage = async () => {
  if (currentPage.value > 1) {
    currentPage.value--;
    const firstItemId =
      selectedType.value === '群组'
        ? String(groupList.value[0].id)
        : String(channelList.value[0].id);
    await getPreviousPage(firstItemId).catch((e) => console.error(e));
  }
};

async function sendGuildChannelMessage(
  message: string,
  options: { id: number } // 只使用 id，因为您说要替换 user_id
) {
  const { data } = await api.accountApiApiUinApiPost(
    props.uin,
    'send_guild_channel_message',
    {
      message,
      ...options,
    }
  );
  return data as { message_id: number }; // 假设返回的数据结构中包含 message_id
}
const sendMessage = async () => {
  try {
    const selectedIds = selectedItems.value.map((item: { id: number }) => ({
      id: item.id,
    }));
    for (const options of selectedIds) {
      const responseData = await sendGuildChannelMessage(
        message.value,
        options
      );
      console.log(`Message sent with ID: ${responseData.message_id}`);
    }
    message.value = ''; // 清空消息输入框
  } catch (e) {
    console.error('发送消息失败:', e);
    // 在这里处理错误，例如显示一个错误消息或者调用一个错误处理函数
  }
};

onMounted(async () => {
  try {
    await fetchDataByType(selectedType.value);
  } catch (e) {
    console.error(e);
  }
});

watch(selectedType, async (newType) => {
  currentPage.value = 1; // 重置页码
  await fetchDataByType(newType).catch((e) => console.error(e));
});
</script>

<style scoped>
/* 样式逻辑，根据您的实际需求添加 */
.selector {
  margin-bottom: 1rem;
}
.message-sender {
  display: flex;
  align-items: center;
  margin-top: 1rem;
}
.q-input {
  flex-grow: 1;
  margin-right: 1rem;
}
.error-message {
  color: red;
  margin: 1rem 0;
}
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 1rem;
}
</style>
