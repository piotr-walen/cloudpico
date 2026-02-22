import { useEffect, useRef } from 'react'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { Reading } from '../schemas/weather'

interface HistoryChartProps {
  readings: Reading[]
}

// Mirror server/static/js/history-chart.js grid config
function createGridConfig(config: { show: boolean; stroke: string; width?: number }) {
  return {
    show: config.show,
    stroke: config.stroke,
    width: config.width ?? 1,
  }
}

// X-axis time formatter: full date+time on first tick of each day, else time only
function formatTimeAxisValues(_u: uPlot, splits: number[]): string[] {
  const seenDates = new Set<string>()
  return splits.map((val) => {
    if (val == null || val === undefined || isNaN(val)) return ''
    const d = new Date(val)
    if (isNaN(d.getTime())) return ''
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const dateKey = `${year}-${month}-${day}`
    const hours = String(d.getHours()).padStart(2, '0')
    const minutes = String(d.getMinutes()).padStart(2, '0')
    if (!seenDates.has(dateKey)) {
      seenDates.add(dateKey)
      return `${day}/${month}/${year} ${hours}:${minutes}`
    }
    return `${hours}:${minutes}`
  })
}

// Legend/tooltip value for time (cursor)
function formatTimeValue(_u: uPlot, val: number): string {
  if (val == null || val === undefined || isNaN(val)) return ''
  const d = new Date(val)
  if (isNaN(d.getTime())) return ''
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${day}/${month}/${year} ${hours}:${minutes}`
}

export function HistoryChart({ readings }: HistoryChartProps) {
  const chartRef = useRef<HTMLDivElement>(null)
  const chartInstanceRef = useRef<uPlot | null>(null)

  useEffect(() => {
    if (!chartRef.current) return

    if (chartInstanceRef.current) {
      chartInstanceRef.current.destroy()
      chartInstanceRef.current = null
    }

    if (readings.length === 0) {
      chartRef.current.innerHTML = '<p class="no-data">No data to display</p>'
      return
    }

    const sortedReadings = [...readings].sort(
      (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime()
    )

    const times: number[] = []
    const temperatures: (number | null)[] = []
    const humidity: (number | null)[] = []
    const pressure: (number | null)[] = []
    const battery: (number | null)[] = []

    sortedReadings.forEach((reading) => {
      times.push(new Date(reading.time).getTime())
      temperatures.push(reading.value ?? null)
      humidity.push(reading.humidityPct > 0 ? reading.humidityPct : null)
      pressure.push(reading.pressureHpa > 0 ? reading.pressureHpa : null)
      battery.push(reading.batteryV > 0 ? reading.batteryV : null)
    })

    // Normalize all series to 0–100 for a single shared y-axis
    const TEMP_RANGE = [-50, 50] as const
    const HUMIDITY_RANGE = [0, 100] as const
    const PRESSURE_RANGE = [900, 1100] as const
    const BATTERY_RANGE = [3, 5] as const

    function normalize(
      val: number | null,
      [lo, hi]: readonly [number, number]
    ): number | null {
      if (val == null) return null
      return ((val - lo) / (hi - lo)) * 100
    }

    const tempNorm = temperatures.map((v) => normalize(v, TEMP_RANGE))
    const humidityNorm = humidity.map((v) => normalize(v, HUMIDITY_RANGE))
    const pressureNorm = pressure.map((v) => normalize(v, PRESSURE_RANGE))
    const batteryNorm = battery.map((v) => normalize(v, BATTERY_RANGE))

    const data: uPlot.AlignedData = [
      times,
      tempNorm,
      humidityNorm,
      pressureNorm,
      batteryNorm,
    ]

    const minTime = times[0]
    const maxTime = times[times.length - 1]
    const padding = (maxTime - minTime) * 0.01

    const gridConfig = {
      xAxis: { show: true, stroke: '#607D8B', width: 1 },
      yAxis: { show: true, stroke: '#607D8B', width: 1 },
    }

    // Legend shows actual values; chart uses normalized 0–100 scale
    const formatRaw =
      (raw: (number | null)[], unit: string) =>
      (_u: uPlot, _val: number, _seriesIdx: number) => {
        const idx = _u.cursor?.idx
        if (idx == null) return '—'
        const v = raw[idx]
        return v != null ? `${v} ${unit}` : '—'
      }

    const opts: uPlot.Options = {
      title: 'Weather Readings',
      width: chartRef.current.offsetWidth || 800,
      height: 400,
      padding: [null, null, 60, 0],
      scales: {
        x: {
          time: true,
          range: [minTime - padding, maxTime + padding] as [number, number],
        },
        y: {
          range: [0, 100] as [number, number],
        },
      },
      series: [
        {
          value: formatTimeValue,
        },
        {
          label: `Temperature (°C)`,
          stroke: '#e74c3c',
          width: 2,
          scale: 'y',
          points: { show: false },
          value: formatRaw(temperatures, '°C'),
        },
        {
          label: `Humidity (%)`,
          stroke: '#3498db',
          width: 2,
          scale: 'y',
          points: { show: false },
          value: formatRaw(humidity, '%'),
        },
        {
          label: `Pressure (hPa)`,
          stroke: '#2ecc71',
          width: 2,
          scale: 'y',
          points: { show: false },
          value: formatRaw(pressure, 'hPa'),
        },
        {
          label: `Battery (V)`,
          stroke: '#f39c12',
          width: 2,
          scale: 'y',
          points: { show: false },
          value: formatRaw(battery, 'V'),
        },
      ],
      axes: [
        {
          scale: 'x',
          stroke: '#607D8B',
          grid: createGridConfig(gridConfig.xAxis),
          rotate: 90,
          values: formatTimeAxisValues,
          ...({ time: true } as { time?: boolean }),
        },
        {
          scale: 'y',
          stroke: '#607D8B',
          grid: createGridConfig(gridConfig.yAxis),
          show: true,
          values: (_u, splits) => splits.map(() => ''),
        },
      ],
      legend: {
        show: true,
        live: true,
      },
      cursor: {
        show: true,
        x: true,
        y: true,
      },
    }

    try {
      const chart = new uPlot(opts, data, chartRef.current)
      chartInstanceRef.current = chart
    } catch (error) {
      console.error('Failed to create chart:', error)
      if (chartRef.current) {
        chartRef.current.innerHTML = '<p class="no-data">Failed to render chart</p>'
      }
    }

    return () => {
      if (chartInstanceRef.current) {
        chartInstanceRef.current.destroy()
        chartInstanceRef.current = null
      }
    }
  }, [readings])

  return <div ref={chartRef} id="history-chart-container" />
}
