import request from './request'

export const filesApi = {
  // 私有空间图片预览签名（编辑器内预览用；提交后查看的页面由后端自动签名）
  signUrls: urls => request.post('/files/sign-urls', { urls })
}
