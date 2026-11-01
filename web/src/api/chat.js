import request from './request'

export const chatApi = {
  conversations: () => request.get('/chat/conversations'),
  messages: (id, page = 0, size = 50) =>
    request.get(`/chat/conversations/${id}/messages`, { params: { page, size } }),
  messagesAfter: (id, afterId, size = 100) =>
    request.get(`/chat/conversations/${id}/messages`, { params: { afterId, size } }),
  send: (conversationId, content) =>
    request.post('/chat/messages', { conversationId, content }),
  markRead: id => request.put(`/chat/conversations/${id}/read`),
  unread: () => request.get('/chat/unread')
}
