import { Link } from 'react-router'
import { useStationsWithLatest } from '../../queries/stations'
import { HistoryIcon } from '../../components/icons/HistoryIcon'
import { formatReadingTime } from '../../utils/formatTime'

function Root() {
  const { data: stations, isLoading, error } = useStationsWithLatest()

  return (
    <main className="main">
      <section className="dashboard">
        <h1>Dashboard</h1>
        <p className="lead">Weather stations and readings.</p>
        <div className="stations-container">
          {isLoading && <div>Loading...</div>}
          {error && <div>Error: {error.message}</div>}
          {stations && stations.length === 0 && (
            <p className="no-data">No stations available</p>
          )}
          {stations && stations.map((station) => (
            <div
              key={station.stationId}
              id={`current-conditions-${station.stationId}`}
              className="current-conditions card"
            >
              <h2 className="card-title">Current conditions</h2>
              <p className="station-name">{station.stationName}</p>
              {station.reading ? (
                <>
                  <p className="reading-value">
                    {station.reading.value.toFixed(1)}°C
                  </p>
                  <p className="reading-extra">
                    <span className="reading-humidity">
                      {Math.round(station.reading.humidityPct)}% humidity
                    </span>
                    <span className="reading-pressure">
                      {Math.round(station.reading.pressureHpa)} hPa
                    </span>
                    {station.reading.batterySocPct != null &&
                      station.reading.batterySocPct >= 0 &&
                      station.reading.batterySocPct <= 100 && (
                        <span className="reading-battery">
                          {station.reading.batteryV > 0
                            ? `${station.reading.batteryV.toFixed(2)}V (${station.reading.batterySocPct.toFixed(0)}%)`
                            : `${station.reading.batterySocPct.toFixed(0)}%`}
                        </span>
                      )}
                  </p>
                  <p
                    className="reading-time"
                    title={station.reading.time}
                    data-time={station.reading.time}
                  >
                    Updated {formatReadingTime(station.reading.time)}
                  </p>
                </>
              ) : (
                <p className="no-data">No recent reading</p>
              )}
            </div>
          ))}
        </div>
      </section>
      <div className="page-nav-buttons">
        <Link to="/history" className="secondary small" role="button">
          <HistoryIcon className="btn-icon" />
          Go to History
        </Link>
      </div>
    </main>
  )
}

export default Root
