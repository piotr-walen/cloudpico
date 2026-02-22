import { useReducer, useEffect, useCallback } from 'react'

/**
 * Custom hook similar to useReducer but with localStorage persistence.
 * 
 * @param key - localStorage key (if undefined, state is not persisted)
 * @param reducer - Reducer function (state, action) => newState
 * @param initialArg - Initial state value or initializer function
 * @param init - Optional initializer function (if initialArg is not a function)
 * @returns [state, dispatch] tuple similar to useReducer
 */
export function useLocalReducer<S, A>(
  key: string | undefined,
  reducer: (state: S, action: A) => S,
  initialArg: S | (() => S),
  init?: (initialArg: S | (() => S)) => S
): [S, (action: A) => void] {
  // Initialize state from localStorage if key is provided, otherwise use initialArg
  const getInitialState = useCallback((): S => {
    if (key === undefined) {
      if (typeof initialArg === 'function') {
        return (initialArg as () => S)()
      }
      return initialArg
    }
    
    try {
      const item = window.localStorage.getItem(key)
      if (item !== null) {
        const parsed = JSON.parse(item) as S
        // If init function is provided, use it to initialize from parsed value
        if (init) {
          return init(parsed)
        }
        return parsed
      }
    } catch (error) {
      console.warn(`Failed to load state from localStorage key "${key}":`, error)
    }
    
    // Fallback to initialArg
    if (typeof initialArg === 'function') {
      return (initialArg as () => S)()
    }
    if (init) {
      return init(initialArg)
    }
    return initialArg
  }, [key, initialArg, init])

  const [state, dispatch] = useReducer(reducer, undefined, getInitialState)

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

  return [state, dispatch]
}
