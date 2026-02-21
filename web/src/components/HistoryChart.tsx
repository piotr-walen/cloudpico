import { useEffect, useRef } from 'react'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'
import type { Reading } from '../schemas/weather'

interface HistoryChartProps {
  readings: Reading[]
  showYAxes: boolean
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

export function HistoryChart({ readings, showYAxes }: HistoryChartProps) {
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

    const data: uPlot.AlignedData = [
      times,
      temperatures,
      humidity,
      pressure,
      battery,
    ]

    const minTime = times[0]
    const maxTime = times[times.length - 1]
    const padding = (maxTime - minTime) * 0.01

    const gridConfig = {
      xAxis: { show: true, stroke: '#607D8B', width: 1 },
      yAxis: { show: showYAxes, stroke: '#607D8B', width: 1 },
    }

    // Options matching server/static/js/history-chart.js
    const opts: uPlot.Options = {
      title: 'Weather Readings',
      width: chartRef.current.offsetWidth || 800,
      height: 400,
      padding: [null, null, 60, null],
      scales: {
        x: {
          time: true,
          range: [minTime - padding, maxTime + padding] as [number, number],
        },
        y: {
          range: [-50, 50] as [number, number],
        },
        y1: {
          range: [0, 100] as [number, number],
        },
        y2: {
          range: [900, 1100] as [number, number],
        },
        y3: {
          range: [3, 5] as [number, number],
        },
      },
      series: [
        {
          value: formatTimeValue,
        },
        {
          label: 'Temperature (°C)',
          stroke: '#e74c3c',
          width: 2,
          scale: 'y',
          points: { show: false },
        },
        {
          label: 'Humidity (%)',
          stroke: '#3498db',
          width: 2,
          scale: 'y1',
          points: { show: false },
        },
        {
          label: 'Pressure (hPa)',
          stroke: '#2ecc71',
          width: 2,
          scale: 'y2',
          points: { show: false },
        },
        {
          label: 'Battery (V)',
          stroke: '#f39c12',
          width: 2,
          scale: 'y3',
          points: { show: false },
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
          show: showYAxes,
          label: 'Temperature (°C)',
        },
        {
          scale: 'y1',
          stroke: '#607D8B',
          grid: createGridConfig({ ...gridConfig.yAxis, show: false }),
          show: showYAxes,
          label: 'Humidity (%)',
        },
        {
          scale: 'y2',
          stroke: '#607D8B',
          grid: createGridConfig({ ...gridConfig.yAxis, show: false }),
          show: showYAxes,
          label: 'Pressure (hPa)',
        },
        {
          scale: 'y3',
          stroke: '#607D8B',
          grid: createGridConfig({ ...gridConfig.yAxis, show: false }),
          show: showYAxes,
          label: 'Battery (V)',
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
  }, [readings, showYAxes])

  return <div ref={chartRef} id="history-chart-container" />
}
