export const campGuides = [
  { name:'柳听风', role:'山村引路人', greeting:'先在「文档」里给树洞建一个家。等你的第一张网页能打开，我们就放上第一条校园留言。' },
  { name:'许望潮', role:'港口领航人', greeting:'写到一半被叫去上课怎么办？给投稿留一份刷新后还能找回的草稿。' },
  { name:'石砚', role:'矿道老匠', greeting:'把帖子接到数据库，请同学用另一台电脑打开同一条讨论。' },
  { name:'沈长安', role:'守关校尉', greeting:'用两个账号试一遍。一个匿名发帖，另一个留言，再让楼主回来查看回复。' },
  { name:'杜衡', role:'机关坊主', greeting:'接到举报后，如何记录处理结果？用测试帖走一遍，再给普通同学看看公开页面。' },
  { name:'白霁', role:'雪岭向导', greeting:'讨论变长以后，试试下一页的加载速度。把帖子隐藏，再确认缓存也及时更新。' },
  { name:'陆观星', role:'观星阁主', greeting:'先部署试运行版。找一台没装开发工具的手机，走一遍体验。' },
  { name:'闻知行', role:'会试掌卷使', greeting:'收集同学试用的意见，修一个影响体验的问题，再分享你的正式作品链接。' }
]

export function guideFor(index) {
  return campGuides[index % campGuides.length]
}
