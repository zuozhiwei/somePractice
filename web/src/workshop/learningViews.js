export const learningViews = [
  { key: 'world', label: '世界地图', description: '在地图上探索关卡' },
  { key: 'project', label: '项目路线', description: '看树洞每个阶段会做成什么' },
  { key: 'story', label: '章节目录', description: '按章节查找关卡' },
  { key: 'tree', label: '技能树', description: '按技能查找练习' }
]

export function learningView(value) {
  return learningViews.find(item => item.key === value)?.key || 'world'
}
