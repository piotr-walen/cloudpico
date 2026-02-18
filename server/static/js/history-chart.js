// History chart initialization using uPlot
(function() {
  'use strict';

  let chart = null;
  // Detect mobile view (typically < 768px width)
  const isMobile = window.innerWidth < 768;
  
  // Read initial showYAxes state from server (driven by server state)
  function getInitialShowYAxes() {
    // First priority: Read from script tag in chart data partial (most reliable, server-driven)
    const showYAxesScript = document.getElementById('history-chart-show-y-axes');
    if (showYAxesScript) {
      try {
        const value = (showYAxesScript.textContent || '').trim();
        return value === 'true';
      } catch (e) {
        console.warn('Failed to parse showYAxes from script tag:', e);
      }
    }
    // Fallback: try to read from the data attribute on the button
    const axesToggle = document.getElementById('axes-toggle');
    if (axesToggle && axesToggle.dataset.showYAxes !== undefined) {
      return axesToggle.dataset.showYAxes === 'true';
    }
    // Default: off on mobile, on on desktop
    return !isMobile;
  }
  
  let showYAxes = getInitialShowYAxes();

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

  // Update chart data without recreating the chart
  function updateChartData() {
    // Check if chart exists and is still valid (DOM element exists)
    if (!chart || !isChartValid()) {
      // Chart doesn't exist or was destroyed by DOM swap, initialize it
      initChart();
      return;
    }

    const readings = parseReadingsData();
    if (readings === null) {
      // Invalid data, but keep existing chart
      return;
    }

    if (readings.length === 0) {
      // No data, but keep chart (might be temporary)
      return;
    }

    const newData = readingsToData(readings);
    if (!newData) {
      return;
    }

    try {
      // Update chart data without recreating
      chart.setData(newData);
      
      // Update axes visibility if it changed
      const newShowYAxes = getInitialShowYAxes();
      if (newShowYAxes !== showYAxes) {
        showYAxes = newShowYAxes;
        // Need to recreate chart if axes visibility changed (structural change)
        initChart();
      }
    } catch (e) {
      console.error('Failed to update chart data:', e);
      // Fallback to full recreation if update fails
      initChart();
    }
  }

  function initChart() {
    // Check if uPlot is available
    if (typeof uPlot === 'undefined') {
      return;
    }
    const container = document.getElementById('history-chart-container');
    const chartArea = document.getElementById('chart-area');
    // Look for the script tag - it's swapped out-of-band, so it should be in the document
    const dataScript = document.getElementById('history-chart-data');
    
    if (!container || !chartArea) {
      return;
    }

    // Parse readings data
    const readings = parseReadingsData();
    
    // If no data script or invalid data, show empty state
    if (readings === null) {
      // The script might not be loaded yet (e.g., on initial page load before partial loads)
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

    // Grid configuration options - experiment with these!
    const gridConfig = {
      // X-axis (horizontal) grid options
      xAxis: {
        show: true,
        stroke: '#607D8B',
        width: 1,
        dash: null, // Set to array like [5, 5] for dashed lines, null for solid
        // filter: null, // Optional function to filter grid lines: (u, splits) => splits
      },
      // Y-axis (vertical) grid options
      yAxis: {
        show: showYAxes,
        stroke: '#607D8B',
        width: 1,
        dash: null, // Set to array like [3, 3] for dashed lines, null for solid
        opacity: 1.0, // 0.0 to 1.0 - affects stroke color opacity
      },
      // Alternative grid styles to try:
      // Subtle grid: { stroke: '#E0E0E0', width: 0.5, opacity: 0.5 }
      // Bold grid: { stroke: '#424242', width: 2, opacity: 1.0 }
      // Dashed grid: { stroke: '#607D8B', width: 1, dash: [5, 5], opacity: 0.7 }
      // Dotted grid: { stroke: '#607D8B', width: 1, dash: [2, 2], opacity: 0.6 }
    };

    // Helper function to create grid config with opacity
    const createGridConfig = (config) => {
      const grid = {
        show: config.show,
        width: config.width,
      };
      
      // Apply stroke color with opacity if specified
      if (config.opacity !== undefined && config.opacity !== 1.0) {
        // Convert hex color to rgba
        const hex = config.stroke.replace('#', '');
        const r = parseInt(hex.substr(0, 2), 16);
        const g = parseInt(hex.substr(2, 2), 16);
        const b = parseInt(hex.substr(4, 2), 16);
        grid.stroke = `rgba(${r}, ${g}, ${b}, ${config.opacity})`;
      } else {
        grid.stroke = config.stroke;
      }
      
      // Apply dash pattern if specified
      if (config.dash && Array.isArray(config.dash)) {
        grid.dash = config.dash;
      }
      
      // Apply filter if specified
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
      
      // Start from the beginning of the first day
      const startDay = new Date(minDate.getFullYear(), minDate.getMonth(), minDate.getDate());
      
      // End at the beginning of the last day (or next day if we want to include it)
      const endDay = new Date(maxDate.getFullYear(), maxDate.getMonth(), maxDate.getDate());
      endDay.setDate(endDay.getDate() + 1); // Include the next day boundary
      
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
      padding: [null, null, 60, null], // [top, right, bottom, left] - extra bottom padding for vertical labels
      scales: {
        x: {
          time: true,
          range: times.length > 0 ? (() => {
            const min = times[0];
            const max = times[times.length - 1];
            // Add small padding (1% on each side)
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
          // First series represents x-axis values in legend
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
          rotate: 90, // Rotate labels vertically
          values: (u, splits) => {
            // Format time labels: show date only on first occurrence of each date
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
              
              // Show date only if this is the first occurrence of this date
              if (!seenDates.has(dateKey)) {
                seenDates.add(dateKey);
                return `${year}-${month}-${day} ${hours}:${minutes}`;
              } else {
                // Show only time for subsequent occurrences on the same date
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
          grid: createGridConfig({ ...gridConfig.yAxis, show: false }), // Hidden grid for secondary axes
          show: showYAxes,
          label: 'Humidity (%)',
        },
        {
          scale: 'y2',
          stroke: '#607D8B',
          grid: createGridConfig({ ...gridConfig.yAxis, show: false }), // Hidden grid for secondary axes
          show: showYAxes,
          label: 'Pressure (hPa)',
        },
        {
          scale: 'y3',
          stroke: '#607D8B',
          grid: createGridConfig({ ...gridConfig.yAxis, show: false }), // Hidden grid for secondary axes
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
            // Draw vertical lines at day boundaries
            // Access times from chart data (u.data[0]) instead of closure variable
            // This ensures we always use the current data, even after updates
            const chartTimes = u.data[0];
            if (!chartTimes || chartTimes.length === 0) {
              return;
            }
            const minTime = chartTimes[0];
            const maxTime = chartTimes[chartTimes.length - 1];
            // Don't draw day boundary lines if time span is greater than 7 days
            const timeSpan = maxTime - minTime;
            const sevenDaysInMs = 7 * 24 * 60 * 60 * 1000;
            if (timeSpan > sevenDaysInMs) {
              return; // Skip drawing day boundary lines for spans > 7 days
            }
            const dayBoundaries = getDayBoundaries(minTime, maxTime);
            
            const { ctx } = u;
            const xScale = u.scales.x;
            const plotTop = u.bbox.top;
            const plotBottom = u.bbox.top + u.bbox.height;
            
            // Detect dark mode
            const isDarkMode = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
            
            ctx.save();
            // Use different colors for light and dark mode
            if (isDarkMode) {
              ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)'; // Light color for dark mode
            } else {
              ctx.strokeStyle = 'rgba(96, 125, 139, 0.3)'; // Darker color for light mode
            }
            ctx.lineWidth = 5;
            ctx.setLineDash([20, 20]); // Dashed lines
            
            dayBoundaries.forEach(timestamp => {
              const xPos = u.valToPos(timestamp, 'x', true);
              
              // Only draw if within the visible range
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

    // Destroy existing chart if recreating (e.g., axes visibility changed)
    if (chart) {
      chart.destroy();
      chart = null;
    }

    // Create chart
    try {
      chart = new uPlot(opts, data, chartArea);
      // Add toggle controls if they don't exist
      addToggleControls(container);
    } catch (e) {
      console.error('Failed to create chart:', e);
      chartArea.innerHTML = '<p class="no-data">Failed to render chart</p>';
      if (chart) {
        chart.destroy();
        chart = null;
      }
    }
  }

  function addToggleControls(container) {
    // Chart container is now in the partial, so controls are already set up by htmx
    // Just update showYAxes from server state
    showYAxes = getInitialShowYAxes();
    
    // Set up htmx event listener for the toggle button
    // Note: Button is recreated on each swap, so we always need to attach the listener
    const axesToggle = document.getElementById('axes-toggle');
    if (axesToggle) {
      // Remove any existing handler to avoid duplicates
      const existingHandler = axesToggle._axesToggleHandler;
      if (existingHandler) {
        axesToggle.removeEventListener('htmx:configRequest', existingHandler);
      }
      
      // Create and attach new handler
      const handler = function(event) {
        // Get current state and toggle it
        const currentState = axesToggle.dataset.showYAxes === 'true';
        const newState = !currentState;
        
        // Add show_y_axes parameter to the request
        event.detail.parameters = event.detail.parameters || {};
        event.detail.parameters.show_y_axes = String(newState);
      };
      axesToggle._axesToggleHandler = handler;
      axesToggle.addEventListener('htmx:configRequest', handler);
    }
  }

  // Listen for htmx swaps - update chart data incrementally
  // Use afterSettle to catch all swaps including out-of-band
  function handleChartUpdate(event) {
    const chartContainer = document.getElementById('history-chart-container');
    if (!chartContainer || !chartContainer.querySelector('#chart-area')) {
      return;
    }

    // Check if chart data script was updated (out-of-band swap replaces the container)
    // If chart exists but its DOM element is gone, we need to recreate it
    if (chart && !isChartValid()) {
      // Chart was destroyed by DOM swap, recreate it
      chart = null;
      initChart();
      return;
    }

    // Check if axes visibility changed before updating state
    const newShowYAxes = getInitialShowYAxes();
    const axesVisibilityChanged = chart && newShowYAxes !== showYAxes;
    
    // Update toggle controls state
    showYAxes = newShowYAxes;
    addToggleControls(chartContainer);

    if (axesVisibilityChanged) {
      // Structural change, need to recreate chart
      initChart();
    } else {
      // Just update data incrementally
      updateChartData();
    }
  }

  // Listen specifically for changes to the history-chart-data script tag
  function handleChartDataUpdate(event) {
    const dataScript = document.getElementById('history-chart-data');
    if (!dataScript) {
      // No data script yet, might be initial load
      return;
    }

    const chartContainer = document.getElementById('history-chart-container');
    if (!chartContainer || !chartContainer.querySelector('#chart-area')) {
      return;
    }

    // Check if chart was invalidated by DOM swap
    if (chart && !isChartValid()) {
      chart = null;
      initChart();
      return;
    }

    // Check if axes visibility changed
    const newShowYAxes = getInitialShowYAxes();
    const axesVisibilityChanged = chart && newShowYAxes !== showYAxes;
    
    // Update state
    showYAxes = newShowYAxes;
    addToggleControls(chartContainer);

    if (axesVisibilityChanged) {
      // Structural change, need to recreate chart
      initChart();
    } else {
      // Just update data incrementally
      updateChartData();
    }
  }
  
  // Listen for htmx swaps on the body (catches all swaps including out-of-band)
  document.body.addEventListener('htmx:afterSettle', handleChartUpdate);
  
  // Also listen on history-container specifically for targeted updates
  const historyContainer = document.getElementById('history-container');
  if (historyContainer) {
    historyContainer.addEventListener('htmx:afterSettle', handleChartUpdate);
  }

  // Set up listeners for the chart data script tag
  function setupChartDataScriptListeners() {
    const chartDataScript = document.getElementById('history-chart-data');
    if (!chartDataScript) {
      return;
    }

    // Remove existing listeners if any (to avoid duplicates)
    if (chartDataScript._chartDataUpdateHandler) {
      chartDataScript.removeEventListener('htmx:afterSwap', chartDataScript._chartDataUpdateHandler);
    }
    if (chartDataScript._chartDataConfigHandler) {
      chartDataScript.removeEventListener('htmx:configRequest', chartDataScript._chartDataConfigHandler);
    }

    // Listen for htmx swaps on the script tag itself
    const updateHandler = function() {
      handleChartDataUpdate();
    };
    chartDataScript._chartDataUpdateHandler = updateHandler;
    chartDataScript.addEventListener('htmx:afterSwap', updateHandler);
    
    // Add show_y_axes parameter to chart data requests
    const configHandler = function(event) {
      // Read show_y_axes from the button's data attribute or script tag
      const showYAxesValue = getInitialShowYAxes();
      event.detail.parameters = event.detail.parameters || {};
      event.detail.parameters.show_y_axes = String(showYAxesValue);
    };
    chartDataScript._chartDataConfigHandler = configHandler;
    chartDataScript.addEventListener('htmx:configRequest', configHandler);
  }

  // Set up listeners initially
  setupChartDataScriptListeners();

  // Re-setup listeners after htmx swaps (script tag might be replaced)
  document.body.addEventListener('htmx:afterSettle', function() {
    setupChartDataScriptListeners();
  });
  
  // Initialize on initial page load (after htmx loads the partial)
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
      
      // Update axes visibility if switching between mobile/desktop
      // Note: We respect server-side state, but can override for mobile/desktop switch
      if (previousIsMobile !== isNowMobile) {
        // Only override if not explicitly set by user (check if button has data attribute)
        const axesToggle = document.querySelector('#axes-toggle');
        if (axesToggle) {
          // On mobile switch, default to hidden; on desktop switch, default to shown
          // But respect user preference if they've explicitly set it
          const currentState = getInitialShowYAxes();
          const newShowYAxes = !isNowMobile; // Off on mobile, on on desktop
          if (newShowYAxes !== showYAxes) {
            showYAxes = newShowYAxes;
            previousIsMobile = isNowMobile;
            axesToggle.textContent = showYAxes ? 'Hide Y-Axes' : 'Show Y-Axes';
            axesToggle.dataset.showYAxes = showYAxes.toString();
            // Reinitialize chart with new axes setting (structural change)
            initChart();
          }
        }
      } else {
        // Just resize the chart if no mobile/desktop switch
        chart.setSize({
          width: document.getElementById('history-chart-container')?.offsetWidth || 800,
          height: 400,
        });
      }
    }, 250);
  });
})();
