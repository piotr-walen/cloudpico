import { useQuery } from '@tanstack/react-query'
import { z } from 'zod'
import { StationsWithLatestSchema, StationSchema } from '../schemas/weather'

const getStationsWithLatest = async () => {
  const url = `${import.meta.env.VITE_API_URL}/stations/with-latest`
  const response = await fetch(url)
  
  if (!response.ok) {
    throw new Error(`Failed to fetch stations: ${response.statusText}`)
  }

  const data = await response.json()
  return StationsWithLatestSchema.parse(data)
}

export const getStations = async () => {
  const url = `${import.meta.env.VITE_API_URL}/stations`
  const response = await fetch(url)
  
  if (!response.ok) {
    throw new Error(`Failed to fetch stations: ${response.statusText}`)
  }

  const data = await response.json()
  return z.array(StationSchema).parse(data)
}

export const useStations = () => {
  return useQuery({
    queryKey: ['stations'],
    queryFn: getStations,
  })
}

export const useStationsWithLatest = () => {
  return useQuery({ 
    queryKey: ['stations', 'with-latest'], 
    queryFn: getStationsWithLatest,
    refetchInterval: 20000, // Refetch every 20 seconds like HTMX
  })
}
