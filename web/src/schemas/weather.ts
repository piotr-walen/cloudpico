import { z } from 'zod'

export const StationSchema = z.object({
  id: z.string(),
  name: z.string(),
})

export const ReadingSchema = z.object({
  stationId: z.string(),
  time: z.string().datetime(),
  value: z.number(), // temperature °C
  humidityPct: z.number(), // 0–100 or 0 if unset
  pressureHpa: z.number(), // hPa or 0 if unset
  batteryV: z.number(), // Battery voltage in volts or 0 if unset
})

export const StationReadingSchema = z.object({
  stationId: z.string(),
  stationName: z.string(),
  reading: ReadingSchema.nullable(),
  batteryPercentage: z.number().optional(),
})

export const StationsWithLatestSchema = z.array(StationReadingSchema)

export type Station = z.infer<typeof StationSchema>
export type Reading = z.infer<typeof ReadingSchema>
export type StationReading = z.infer<typeof StationReadingSchema>
