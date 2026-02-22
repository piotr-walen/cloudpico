import { useQuery } from '@tanstack/react-query'
import { z } from 'zod'
import { ReadingSchema } from '../schemas/weather'

const HistoryReadingsResponseSchema = z.object({
  readings: z.array(ReadingSchema).nullable().transform((val) => val ?? []),
  page: z.number(),
  limit: z.number(),
  total: z.number(),
  totalPages: z.number(),
  hasPrev: z.boolean(),
  hasNext: z.boolean(),
})

export type HistoryReadingsResponse = z.infer<typeof HistoryReadingsResponseSchema>

const getHistoryReadings = async (
  stationId: string,
  range: string,
  page: number = 1,
  limit: number = 20
): Promise<HistoryReadingsResponse> => {
  const path = `${import.meta.env.VITE_API_URL}/stations/${stationId}/history/readings`
  const url = `${path}?${new URLSearchParams({ range, page: page.toString(), limit: limit.toString() })}`
  const response = await fetch(url)

  if (!response.ok) {
    throw new Error(`Failed to fetch history readings: ${response.statusText}`)
  }

  const data = await response.json()
  return HistoryReadingsResponseSchema.parse(data)
}

const getHistoryChart = async (
  stationId: string,
  range: string
): Promise<z.infer<typeof ReadingSchema>[]> => {
  const path = `${import.meta.env.VITE_API_URL}/stations/${stationId}/history/chart`
  const url = `${path}?${new URLSearchParams({ range })}`
  const response = await fetch(url)

  if (!response.ok) {
    throw new Error(`Failed to fetch chart data: ${response.statusText}`)
  }

  const data = await response.json()
  return z.array(ReadingSchema).parse(data)
}

export const useHistoryReadings = (
  stationId: string | null,
  range: string,
  page: number = 1,
  limit: number = 20
) => {
  return useQuery({
    queryKey: ['history', 'readings', stationId, range, page, limit],
    queryFn: () => getHistoryReadings(stationId!, range, page, limit),
    enabled: !!stationId,
  })
}

export const useHistoryChart = (stationId: string | null, range: string) => {
  return useQuery({
    queryKey: ['history', 'chart', stationId, range],
    queryFn: () => getHistoryChart(stationId!, range),
    enabled: !!stationId,
  })
}
