// History chart initialization using uPlot
(function() {
  'use strict';

  let chart = null;
  // Detect mobile view (typically < 768px width)
  const isMobile = window.innerWidth < 768;
  
  // Read initial showYAxes state from server (cookie)
  function getInitialShowYAxes() {
    // Read from script tag in partial (most reliable)
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

    // Destroy existing chart if any
    if (chart) {
      chart.destroy();
      chart = null;
    }
    
    // Clear chart area
    chartArea.innerHTML = '';

    // If no data script, show empty state
    if (!dataScript) {
      // The script might not be loaded yet (e.g., on initial page load before partial loads)
      // Show loading message - the chart will initialize when the partial loads via htmx
      chartArea.innerHTML = '<p class="no-data">Loading chart data…</p>';
      return;
    }

    // Parse readings data
    let readings;
    try {
      const jsonText = (dataScript.textContent || '').trim();
      if (!jsonText) {
        readings = [];
      } else {
        readings = JSON.parse(jsonText);
      }
    } catch (e) {
      console.error('Failed to parse chart data:', e, 'Content:', dataScript.textContent);
      chartArea.innerHTML = '<p class="no-data">Failed to parse chart data</p>';
      return;
    }

    // Ensure readings is an array
    if (!Array.isArray(readings)) {
      console.error('Chart data is not an array. Type:', typeof readings);
      // Try to convert if it's an array-like object
      if (readings && typeof readings === 'object' && readings.length !== undefined) {
        readings = Array.from(readings);
      } else {
        chartArea.innerHTML = '<p class="no-data">Invalid chart data format</p>';
        return;
      }
    }

    if (readings.length === 0) {
      chartArea.innerHTML = '<p class="no-data">No data to display</p>';
      return;
    }

    // Sort readings by time (oldest first) for proper time series display
    readings.sort((a, b) => new Date(a.time) - new Date(b.time));

    // Prepare data arrays for uPlot
    // uPlot expects: [time, series1, series2, ...]
    const times = [];
    const temperatures = [];
    const humidity = [];
    const pressure = [];
    const battery = [];

    readings.forEach(reading => {
      const timestamp = new Date(reading.time).getTime();
      times.push(timestamp);
      temperatures.push(reading.value ?? null);
      humidity.push(reading.humidityPct > 0 ? reading.humidityPct : null);
      pressure.push(reading.pressureHpa > 0 ? reading.pressureHpa : null);
      battery.push(reading.batteryV > 0 ? reading.batteryV : null);
    });

    const data = [
      times,
      temperatures,
      humidity,
      pressure,
      battery
    ];

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
            const minTime = times[0];
            const maxTime = times[times.length - 1];
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

    // Create chart
    try {
      chart = new uPlot(opts, data, chartArea);
      // Add toggle controls if they don't exist
      addToggleControls(container);
    } catch (e) {
      console.error('Failed to create chart:', e);
      chartArea.innerHTML = '<p class="no-data">Failed to render chart</p>';
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

  // Listen for swaps - htmx handles everything
  // Use afterSettle to catch all swaps including out-of-band
  function reinitChart() {
    const chartContainer = document.getElementById('history-chart-container');
    if (chartContainer && chartContainer.querySelector('#chart-area')) {
      showYAxes = getInitialShowYAxes();
      addToggleControls(chartContainer);
      initChart();
    }
  }
  
  document.body.addEventListener('htmx:afterSettle', reinitChart);
  
  // Also listen on history-container specifically
  const historyContainer = document.getElementById('history-container');
  if (historyContainer) {
    historyContainer.addEventListener('htmx:afterSettle', reinitChart);
  }
  
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
          showYAxes = !isNowMobile; // Off on mobile, on on desktop
          previousIsMobile = isNowMobile;
          axesToggle.textContent = showYAxes ? 'Hide Y-Axes' : 'Show Y-Axes';
          axesToggle.dataset.showYAxes = showYAxes.toString();
        }
        // Reinitialize chart with new axes setting
        initChart();
      } else if (chart) {
        // Just resize the chart if no mobile/desktop switch
        chart.setSize({
          width: document.getElementById('history-chart-container')?.offsetWidth || 800,
          height: 400,
        });
      }
    }, 250);
  });
})();
