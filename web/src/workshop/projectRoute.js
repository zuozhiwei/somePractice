import { chapters } from '../../../internal/curriculum/catalog.json'

const icons = ['Monitor', 'Collection', 'Coin', 'User', 'Checked', 'TrendCharts', 'Connection', 'Flag']
const tones = ['mint', 'sand', 'lilac', 'peach']

export const projectStages = chapters.map((chapter, index) => ({
  id: chapter.id,
  name: chapter.name,
  result: chapter.story || chapter.description,
  icon: icons[index % icons.length],
  tone: tones[index % tones.length]
}))
