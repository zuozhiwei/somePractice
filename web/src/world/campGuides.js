export const campGuides = [
  { name:'柳听风', role:'山村引路人', greeting:'初入江湖，先走稳脚下的一步。我替你留了一份入门任务。' },
  { name:'许望潮', role:'港口领航人', greeting:'渡船已经靠岸。把手里的活做扎实，才能驶向下一程。' },
  { name:'石砚', role:'矿道老匠', greeting:'地面之下也有章法。带着问题进矿，再带着证据出来。' },
  { name:'沈长安', role:'守关校尉', greeting:'过关要看真本事。先看清任务要求，再来交你的答卷。' },
  { name:'杜衡', role:'机关坊主', greeting:'来试试你的手艺。把眼前这件事做对，再琢磨它能不能做得更好。' },
  { name:'白霁', role:'雪岭向导', greeting:'越往高处越要留好路标。沿途遇到的问题，也值得写进记录。' },
  { name:'陆观星', role:'观星阁主', greeting:'走到此处，已经能独当一面。打开任务，看看这程该怎样走。' },
  { name:'闻知行', role:'会试掌卷使', greeting:'这里是会试考场。先读题目和验收要求，再用自己的项目、测试记录与讲解作答。' }
]

export function guideFor(index) {
  return campGuides[index % campGuides.length]
}
