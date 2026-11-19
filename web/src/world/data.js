export const TILE = 32

export function mentorLines(map, level, tasks, mentorName) {
  const lines = [
    `${mentorName}：来了，学徒。这一带想闯出名堂，先从「${level.name}」开始。`,
    `${mentorName}：这一关的关键是——${level.description}。`
  ]
  const todo = tasks.filter(t => t.submissionStatus !== 'APPROVED')
  if (todo.length) {
    lines.push(`${mentorName}：眼下有 ${todo.length} 件作业要做：「${todo[0].title}」，做完把它交给我。`)
  } else {
    lines.push(`${mentorName}：好小子，这关的作业全交齐了。去下一个师父那儿吧。`)
  }
  lines.push(`${mentorName}：要交作业就点下面的「提交作业」。截图、笔记都行，我看过才算数。`)
  return lines
}
