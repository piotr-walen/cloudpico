// History chart initialization using uPlot
(function() {
  'use strict';

  let chart = null;
  // Detect mobile view (typically < 768px width)
  const isMobile = window.innerWidth < 768;
  
  // Y-axes visibility state - stored in localStorage, default based on device
  const STORAGE_KEY = 'history-chart-show-y-axes';
  function getShowYAxes() {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored !== null) {
      return stored === 'true';
    }
    // Default: off on mobile, on on desktop
    return !isMobile;
  }
  
  function setShowYAxes(value) {
    localStorage.setItem(STORAGE_KEY, String(value));
  }
  
  let showYAxes = getShowYAxes();

  // Parse readings data from script tag
  function parseReadingsData() {
    const dataScript = document.getElementById('history-chart-data');
    if (!dataScript) {
      return null;
    }

    try {
      const jsonText = (dataScript.textContent || '').trim();
      if (!jsonText) {
        return [];
      }
      const readings = JSON.parse(jsonText);
      
      // Ensure readings is an array
      if (!Array.isArray(readings)) {
        console.error('Chart data is not an array. Type:', typeof readings);
        if (readings && typeof readings === 'object' && readings.length !== undefined) {
          return Array.from(readings);
        }
        return null;
      }
      
      return readings;
    } catch (e) {
      console.error('Failed to parse chart data:', e, 'Content:', dataScript?.textContent);
      return null;
    }
  }

  // Convert readings to uPlot data format
  function readingsToData(readings) {
    if (!readings || readings.length === 0) {
      return null;
    }

    // Sort readings by time (oldest first) for proper time series display
    const sortedReadings = [...readings].sort((a, b) => new Date(a.time) - new Date(b.time));

    // Prepare data arrays for uPlot
    // uPlot expects: [time, series1, series2, ...]
    const times = [];
    const temperatures = [];
    const humidity = [];
    const pressure = [];
    const battery = [];

    sortedReadings.forEach(reading => {
      const timestamp = new Date(reading.time).getTime();
      times.push(timestamp);
      temperatures.push(reading.value ?? null);
      humidity.push(reading.humidityPct > 0 ? reading.humidityPct : null);
      pressure.push(reading.pressureHpa > 0 ? reading.pressureHpa : null);
      battery.push(reading.batteryV > 0 ? reading.batteryV : null);
    });

    return [
      times,
      temperatures,
      humidity,
      pressure,
      battery
    ];
  }

  // Check if chart is still valid (its DOM element exists)
  function isChartValid() {
    if (!chart) {
      return false;
    }
    // Check if the chart's root element still exists in the DOM
    const chartRoot = chart.root;
    return chartRoot && document.body.contains(chartRoot);
  }

  function initChart() {
    // Check if uPlot is available
    if (typeof uPlot === 'undefined') {
      return;
    }
    const container = document.getElementById('history-chart-container');
    const chartArea = document.getElementById('chart-area');
    const dataScript = document.getElementById('history-chart-data');
    
    if (!container || !chartArea) {
      return;
    }

    // Parse readings data
    const readings = parseReadingsData();
    
    // If no data script or invalid data, show empty state
    if (readings === null) {
      if (!dataScript) {
        chartArea.innerHTML = '<p class="no-data">Loading chart data…</p>';
      } else {
        chartArea.innerHTML = '<p class="no-data">Failed to parse chart data</p>';
      }
      // Destroy existing chart if any
      if (chart) {
        chart.destroy();
        chart = null;
      }
      return;
    }

    if (readings.length === 0) {
      chartArea.innerHTML = '<p class="no-data">No data to display</p>';
      // Destroy existing chart if any
      if (chart) {
        chart.destroy();
        chart = null;
      }
      return;
    }

    // Clear any error messages
    chartArea.innerHTML = '';

    const data = readingsToData(readings);
    if (!data) {
      chartArea.innerHTML = '<p class="no-data">Invalid chart data format</p>';
      if (chart) {
        chart.destroy();
        chart = null;
      }
      return;
    }

    // Extract times array from data (first element)
    const times = data[0];

    // Grid configuration
    const gridConfig = {
      xAxis: {
        show: true,
        stroke: '#607D8B',
        width: 1,
        dash: null,
      },
      yAxis: {
        show: showYAxes,
        stroke: '#607D8B',
        width: 1,
        dash: null,
        opacity: 1.0,
      },
    };

    // Helper function to create grid config with opacity
    const createGridConfig = (config) => {
      const grid = {
        show: config.show,
        width: config.width,
      };
      
      // Apply stroke color with opacity if specified
      if (config.opacity !== undefined && config.opacity !== 1.0) {
        const hex = config.stroke.replace('#', '');
        const r = parseInt(hex.substr(0, 2), 16);
        const g = parseInt(hex.substr(2, 2), 16);
        const b = parseInt(hex.substr(4, 2), 16);
        grid.stroke = `rgba(${r}, ${g}, ${b}, ${config.opacity})`;
      } else {
        grid.stroke = config.stroke;
      }
      
      if (config.dash && Array.isArray(config.dash)) {
        grid.dash = config.dash;
      }
      
      if (config.filter) {
        grid.filter = config.filter;
      }
      
      return grid;
    };

    // Calculate day boundaries (midnight timestamps) for vertical grid lines
    const getDayBoundaries = (minTime, maxTime) => {
      const boundaries = [];
      const minDate = new Date(minTime);
      const maxDate = new Date(maxTime);
      
      const startDay = new Date(minDate.getFullYear(), minDate.getMonth(), minDate.getDate());
      const endDay = new Date(maxDate.getFullYear(), maxDate.getMonth(), maxDate.getDate());
      endDay.setDate(endDay.getDate() + 1);
      
      let currentDay = new Date(startDay);
      while (currentDay <= endDay) {
        boundaries.push(currentDay.getTime());
        currentDay.setDate(currentDay.getDate() + 1);
      }
      
      return boundaries;
    };

    // Chart options
    const opts = {
      title: 'Weather Readings',
      width: chartArea.offsetWidth || container.offsetWidth || 800,
      height: 400,
      padding: [null, null, 60, null],
      scales: {
        x: {
          time: true,
          range: times.length > 0 ? (() => {
            const min = times[0];
            const max = times[times.length - 1];
            const padding = (max - min) * 0.01;
            return [min - padding, max + padding];
          })() : undefined,
        },
        y: {
          label: 'Temperature (°C)',
          side: 1,
          range: [-50, 50],
        },
        y1: {
          label: 'Humidity (%)',
          side: 2,
          range: [0, 100],
        },
        y2: {
          label: 'Pressure (hPa)',
          side: 1,
          range: [900, 1100],
        },
        y3: {
          label: 'Battery (V)',
          side: 2,
          range: [3, 5],
        },
      },
      series: [
        {
          value: (u, val) => {
            if (val == null || val === undefined || isNaN(val)) {
              return '';
            }
            const d = new Date(val);
            if (isNaN(d.getTime())) {
              return '';
            }
            const year = d.getFullYear();
            const month = String(d.getMonth() + 1).padStart(2, '0');
            const day = String(d.getDate()).padStart(2, '0');
            const hours = String(d.getHours()).padStart(2, '0');
            const minutes = String(d.getMinutes()).padStart(2, '0');
            return `${year}-${month}-${day} ${hours}:${minutes}`;
          },
        },
        {
          label: 'Temperature (°C)',
          stroke: '#e74c3c',
          width: 2,
          scale: 'y',
        },
        {
          label: 'Humidity (%)',
          stroke: '#3498db',
          width: 2,
          scale: 'y1',
        },
        {
          label: 'Pressure (hPa)',
          stroke: '#2ecc71',
          width: 2,
          scale: 'y2',
        },
        {
          label: 'Battery (V)',
          stroke: '#f39c12',
          width: 2,
          scale: 'y3',
        },
      ],
      axes: [
        {
          stroke: '#607D8B',
          grid: createGridConfig(gridConfig.xAxis),
          time: true,
          rotate: 90,
          values: (u, splits) => {
            const seenDates = new Set();
            return splits.map((val) => {
              if (val == null || val === undefined || isNaN(val)) {
                return '';
              }
              
              const d = new Date(val);
              
              if (isNaN(d.getTime())) {
                return '';
              }
              
              const year = d.getFullYear();
              const month = String(d.getMonth() + 1).padStart(2, '0');
              const day = String(d.getDate()).padStart(2, '0');
              const dateKey = `${year}-${month}-${day}`;
              const hours = String(d.getHours()).padStart(2, '0');
              const minutes = String(d.getMinutes()).padStart(2, '0');
              
              if (!seenDates.has(dateKey)) {
                seenDates.add(dateKey);
                return `${year}-${month}-${day} ${hours}:${minutes}`;
              } else {
                return `${hours}:${minutes}`;
              }
            });
          },
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
        live: true,
      },
      cursor: {
        show: true,
        x: true,
        y: true,
      },
      hooks: {
        draw: [
          (u) => {
            const chartTimes = u.data[0];
            if (!chartTimes || chartTimes.length === 0) {
              return;
            }
            const minTime = chartTimes[0];
            const maxTime = chartTimes[chartTimes.length - 1];
            const timeSpan = maxTime - minTime;
            const sevenDaysInMs = 7 * 24 * 60 * 60 * 1000;
            if (timeSpan > sevenDaysInMs) {
              return;
            }
            const dayBoundaries = getDayBoundaries(minTime, maxTime);
            
            const { ctx } = u;
            const xScale = u.scales.x;
            const plotTop = u.bbox.top;
            const plotBottom = u.bbox.top + u.bbox.height;
            
            const isDarkMode = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
            
            ctx.save();
            if (isDarkMode) {
              ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
            } else {
              ctx.strokeStyle = 'rgba(96, 125, 139, 0.3)';
            }
            ctx.lineWidth = 5;
            ctx.setLineDash([20, 20]);
            
            dayBoundaries.forEach(timestamp => {
              const xPos = u.valToPos(timestamp, 'x', true);
              
              if (xPos >= u.bbox.left && xPos <= u.bbox.left + u.bbox.width) {
                ctx.beginPath();
                ctx.moveTo(xPos, plotTop);
                ctx.lineTo(xPos, plotBottom);
                ctx.stroke();
              }
            });
            
            ctx.restore();
          }
        ]
      }
    };

    // Destroy existing chart if recreating
    if (chart) {
      chart.destroy();
      chart = null;
    }

    // Create chart
    try {
      chart = new uPlot(opts, data, chartArea);
      // Set up toggle button
      setupToggleButton();
    } catch (e) {
      console.error('Failed to create chart:', e);
      chartArea.innerHTML = '<p class="no-data">Failed to render chart</p>';
      if (chart) {
        chart.destroy();
        chart = null;
      }
    }
  }

  // Set up the toggle button (pure JS, no htmx)
  function setupToggleButton() {
    const axesToggle = document.getElementById('axes-toggle');
    if (!axesToggle) {
      return;
    }

    // Remove any existing handlers
    if (axesToggle._toggleHandler) {
      axesToggle.removeEventListener('click', axesToggle._toggleHandler);
    }

    // Update button text
    axesToggle.textContent = showYAxes ? 'Hide Y-Axes' : 'Show Y-Axes';
    axesToggle.dataset.showYAxes = String(showYAxes);

    // Add click handler
    const handler = function(e) {
      e.preventDefault();
      e.stopPropagation();
      
      // Toggle state
      showYAxes = !showYAxes;
      setShowYAxes(showYAxes);
      
      // Update button
      axesToggle.textContent = showYAxes ? 'Hide Y-Axes' : 'Show Y-Axes';
      axesToggle.dataset.showYAxes = String(showYAxes);
      
      // Recreate chart with new axes visibility
      initChart();
    };
    
    axesToggle._toggleHandler = handler;
    axesToggle.addEventListener('click', handler);
    
    // Remove htmx attributes to prevent server round-trip
    axesToggle.removeAttribute('hx-get');
    axesToggle.removeAttribute('hx-target');
    axesToggle.removeAttribute('hx-swap');
    axesToggle.removeAttribute('hx-include');
  }

  // Handle chart updates when data changes
  function handleChartUpdate() {
    // Small delay to ensure DOM is settled
    setTimeout(() => {
      const chartContainer = document.getElementById('history-chart-container');
      const chartArea = document.getElementById('chart-area');
      
      if (!chartContainer || !chartArea) {
        return;
      }

      // Check if chart was invalidated by DOM swap
      if (chart && !isChartValid()) {
        chart = null;
      }

      // Always recreate chart on data changes to ensure x-axis updates correctly
      initChart();
    }, 100);
  }
  
  // Listen for htmx swaps
  document.body.addEventListener('htmx:afterSettle', handleChartUpdate);
  
  // Listen for chart data script tag swaps
  function setupChartDataScriptListeners() {
    const chartDataScript = document.getElementById('history-chart-data');
    if (!chartDataScript) {
      return;
    }

    // Remove existing listeners
    if (chartDataScript._chartDataUpdateHandler) {
      chartDataScript.removeEventListener('htmx:afterSwap', chartDataScript._chartDataUpdateHandler);
      chartDataScript.removeEventListener('htmx:afterSettle', chartDataScript._chartDataUpdateHandler);
    }

    const updateHandler = handleChartUpdate;
    chartDataScript._chartDataUpdateHandler = updateHandler;
    chartDataScript.addEventListener('htmx:afterSwap', updateHandler);
    chartDataScript.addEventListener('htmx:afterSettle', updateHandler);
  }

  // Set up listeners initially
  setupChartDataScriptListeners();

  // Re-setup listeners after htmx swaps
  document.body.addEventListener('htmx:afterSettle', function() {
    setupChartDataScriptListeners();
  });
  
  // Initialize on page load
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      setTimeout(initChart, 200);
    });
  } else {
    setTimeout(initChart, 200);
  }

  // Handle window resize
  let resizeTimeout;
  let previousIsMobile = window.innerWidth < 768;
  window.addEventListener('resize', function() {
    clearTimeout(resizeTimeout);
    resizeTimeout = setTimeout(function() {
      if (!chart) {
        return;
      }

      const isNowMobile = window.innerWidth < 768;
      
      if (previousIsMobile !== isNowMobile) {
        // Update axes visibility for mobile/desktop switch
        const newShowYAxes = !isNowMobile;
        if (newShowYAxes !== showYAxes) {
          showYAxes = newShowYAxes;
          setShowYAxes(showYAxes);
          previousIsMobile = isNowMobile;
          setupToggleButton();
          initChart();
        }
      } else {
        // Just resize the chart
        chart.setSize({
          width: document.getElementById('history-chart-container')?.offsetWidth || 800,
          height: 400,
        });
      }
    }, 250);
  });
})();
