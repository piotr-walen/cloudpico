import { useState, useEffect, useCallback } from 'react'

/**
 * Custom hook for state management that can be easily extended with localStorage persistence.
 * Similar to useMutation pattern - provides state and setter with optional persistence.
 * 
 * @param key - localStorage key (if undefined, state is not persisted)
 * @param initialValue - Initial state value
 * @returns [state, setState] tuple similar to useState
 */
export function useLocalState<T>(
  key: string | undefined,
  initialValue: T
): [T, (value: T | ((prev: T) => T)) => void] {
  // Initialize state from localStorage if key is provided, otherwise use initialValue
  const [state, setState] = useState<T>(() => {
    if (key === undefined) {
      return initialValue
    }
    
    try {
      const item = window.localStorage.getItem(key)
      if (item !== null) {
        return JSON.parse(item) as T
      }
    } catch (error) {
      console.warn(`Failed to load state from localStorage key "${key}":`, error)
    }
    
    return initialValue
  })

  // Persist state to localStorage whenever it changes (if key is provided)
  useEffect(() => {
    if (key === undefined) {
      return
    }
    
    try {
      window.localStorage.setItem(key, JSON.stringify(state))
    } catch (error) {
      console.warn(`Failed to save state to localStorage key "${key}":`, error)
    }
  }, [key, state])

  // Wrapper for setState that handles both direct values and updater functions
  const setStateWithPersistence = useCallback(
    (value: T | ((prev: T) => T)) => {
      setState((prev) => {
        if (typeof value === 'function') {
          return (value as (prev: T) => T)(prev)
        }
        return value
      })
    },
    []
  )

  return [state, setStateWithPersistence]
}
