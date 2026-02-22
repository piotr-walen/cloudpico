import { useQuery } from '@tanstack/react-query'
import z from 'zod'

const healthzSchema = z.object({
    status: z.enum(['ok', 'error']),
    mqtt: z.enum(['connected', 'disconnected']),
})

const getHealthz = async () => {
    const url = `${import.meta.env.VITE_API_URL}/healthz`
    const response = await fetch(url)
    
    if (!response.ok) {
        let errorMessage = `HTTP ${response.status}: ${response.statusText}`
        try {
            const errorData = await response.json()
            if (errorData.message) {
                errorMessage = errorData.message
            }
        } catch {
            // If response is not JSON, use the status text
            const text = await response.text()
            if (text) {
                errorMessage = text
            }
        }
        throw new Error(errorMessage)
    }

    const contentType = response.headers.get('content-type')
    if (!contentType || !contentType.includes('application/json')) {
        const text = await response.text()
        throw new Error(`Expected JSON but got ${contentType}. Response: ${text.substring(0, 100)}`)
    }

    const data = await response.json()
    return healthzSchema.parse(data)
}

export const useHealthz = () => {
    return useQuery({ queryKey: ['healthz'], queryFn: getHealthz })
}