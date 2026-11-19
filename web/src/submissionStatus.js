const labels = {
  NONE: '待提交',
  PENDING: '待审核',
  AI_REVIEWED: '待人工审核',
  APPROVED: '已通过',
  REJECTED: '未通过'
}

export function submissionStatusText(status) {
  return labels[status] || '未知状态'
}

export function awaitingReview(status) {
  return status === 'PENDING' || status === 'AI_REVIEWED'
}

export function submissionActionText(status) {
  if (status === 'APPROVED') return '已完成'
  if (status === 'REJECTED') return '重新提交'
  if (awaitingReview(status)) return '补充作业'
  return '提交作业'
}
