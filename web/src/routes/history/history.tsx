import React from 'react'
import { useHistoryChart } from '../../queries/history'
import { useStations } from '../../queries/stations'
import { useLocalReducer } from '../../hooks/useLocalReducer'
import { HistoryChart } from '../../components/HistoryChart'
import { Link } from 'react-router'
import { CloudIcon } from '../../components/icons/CloudIcon'

const HISTORY_RANGES = [
  { key: '1h', label: 'Last 1 hour' },
  { key: '6h', label: 'Last 6 hours' },
  { key: '24h', label: 'Last 24 hours' },
  { key: '3d', label: 'Last 3 days' },
  { key: '7d', label: 'Last 7 days' },
  { key: '30d', label: 'Last 30 days' },
] as const

const DEFAULT_RANGE = '24h'

interface HistoryState {
  stationId: string | null
  range: string
}

type HistoryAction =
  | { type: 'SET_STATION'; stationId: string | null }
  | { type: 'SET_RANGE'; range: string }

function historyReducer(
  state: HistoryState,
  action: HistoryAction
): HistoryState {
  switch (action.type) {
    case 'SET_STATION':
      return { ...state, stationId: action.stationId }
    case 'SET_RANGE':
      return { ...state, range: action.range }
    default:
      return state
  }
}

function History() {
  const { data: stations, isLoading: stationsLoading, error: stationsError } = useStations()

  // Get initial stationId - prefer first station if available
  const initialStationId = React.useMemo(() => {
    if (stations && stations.length > 0) {
      return stations[0].id
    }
    return null
  }, [stations])

  const [state, dispatch] = useLocalReducer(
    'history-state',
    historyReducer,
    {
      stationId: initialStationId,
      range: DEFAULT_RANGE,
    },
    (initial): HistoryState => {
      // Initialize from localStorage, but ensure stationId is valid
      const validated: HistoryState = { ...(initial as HistoryState) }
      
      // Validate stationId against current stations list
      if (
        validated.stationId &&
        stations &&
        !stations.some((s) => s.id === validated.stationId)
      ) {
        validated.stationId = stations.length > 0 ? stations[0].id : null
      }
      
      // If no stationId set and stations are available, use first one
      if (!validated.stationId && stations && stations.length > 0) {
        validated.stationId = stations[0].id
      }
      
      return validated
    }
  )

  // Update stationId when stations load if it's not set
  React.useEffect(() => {
    if (stations && stations.length > 0 && !state.stationId) {
      dispatch({ type: 'SET_STATION', stationId: stations[0].id })
    }
  }, [stations, state.stationId, dispatch])

  const { data: chartData, isLoading: chartLoading, error: chartError } = useHistoryChart(
    state.stationId,
    state.range
  )

  return (
    <main className="main">
      <section className="history">
        <h1>History</h1>
        <p className="lead">View historical weather station readings.</p>

        <div className="history-controls">
          <div className="form-group">
            <label htmlFor="station-select">Station:</label>
            <select
              id="station-select"
              value={state.stationId || ''}
              onChange={(e) =>
                dispatch({
                  type: 'SET_STATION',
                  stationId: e.target.value || null,
                })
              }
            >
              {stations?.map((station) => (
                <option key={station.id} value={station.id}>
                  {station.name}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="range-select">Time Range:</label>
            <select
              id="range-select"
              value={state.range}
              onChange={(e) =>
                dispatch({ type: 'SET_RANGE', range: e.target.value })
              }
            >
              {HISTORY_RANGES.map((range) => (
                <option key={range.key} value={range.key}>
                  {range.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {stationsLoading && <div>Loading stations...</div>}
        {stationsError && (
          <div className="error">
            Error loading stations: {stationsError.message}
          </div>
        )}
        {!stationsLoading && !stationsError && (!stations || stations.length === 0) && (
          <div className="no-data">No stations available</div>
        )}

        {!state.stationId && (
          <div className="no-data">Please select a station</div>
        )}

        {chartLoading && state.stationId && <div>Loading chart...</div>}
        {chartError && (
          <div className="error">
            Error loading chart: {chartError.message}
          </div>
        )}

        {chartData && chartData.length > 0 && (
          <div className="chart-container">
            <HistoryChart readings={chartData} />
          </div>
        )}

        {chartData && chartData.length === 0 && state.stationId && (
          <p className="no-data">No readings available for the selected time range</p>
        )}

        <div className="page-nav-buttons">
          <Link to="/" className="secondary small" role="button">
            <CloudIcon className="btn-icon" />
            Back to Dashboard
          </Link>
        </div>
      </section>
    </main>
  )
}

export default History
