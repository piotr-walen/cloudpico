// History chart initialization using uPlot
(function() {
  'use strict';

  let chart: uPlotTypes.UPlot | null = null;
  // Detect mobile view (typically < 768px width)
  const isMobile = window.innerWidth < 768;
  
  // Read initial showYAxes state from server (cookie)
  function getInitialShowYAxes(): boolean {
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

  interface Reading {
    time: string;
    value?: number | null;
    humidityPct?: number;
    pressureHpa?: number;
    batteryV?: number;
  }

  function initChart(): void {
    // Check if uPlot is available
    if (typeof (window as any).uPlot === 'undefined') {
      return;
    }
    const uPlotLib = (window as any).uPlot as typeof uPlotTypes.UPlot;
    
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
    let readings: Reading[];
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
      if (readings && typeof readings === 'object' && (readings as any).length !== undefined) {
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
    readings.sort((a, b) => new Date(a.time).getTime() - new Date(b.time).getTime());

    // Prepare data arrays for uPlot
    // uPlot expects: [time, series1, series2, ...]
    const times: number[] = [];
    const temperatures: (number | null)[] = [];
    const humidity: (number | null)[] = [];
    const pressure: (number | null)[] = [];
    const battery: (number | null)[] = [];

    readings.forEach(reading => {
      const timestamp = new Date(reading.time).getTime();
      times.push(timestamp);
      temperatures.push(reading.value ?? null);
      humidity.push(reading.humidityPct && reading.humidityPct > 0 ? reading.humidityPct : null);
      pressure.push(reading.pressureHpa && reading.pressureHpa > 0 ? reading.pressureHpa : null);
      battery.push(reading.batteryV && reading.batteryV > 0 ? reading.batteryV : null);
    });

    const data: uPlotTypes.AlignedData = [
      times,
      temperatures,
      humidity,
      pressure,
      battery
    ];

    // Import the date formatter function
    // Note: In the compiled JS, this will be a regular import
    // For now, we'll inline it here but use the extracted function in tests
    const formatTimeLabels = (splits: number[], minTimestamp: number, maxTimestamp: number): string[] => {
      const range = maxTimestamp - minTimestamp;
      const hours = range / (1000 * 60 * 60);
      const days = hours / 24;

      return splits.map((val, index) => {
        // Ensure val is a valid number (timestamp in milliseconds)
        const timestamp = typeof val === 'number' && !isNaN(val) ? val : Number(val);
        
        // Validate timestamp is reasonable (between 1970 and 2100)
        if (isNaN(timestamp) || timestamp < 0 || timestamp > 4102444800000) {
          console.warn(`Invalid timestamp: ${val}`);
          return '';
        }

        const d = new Date(timestamp);
        
        // Validate date object
        if (isNaN(d.getTime())) {
          console.warn(`Invalid date created from timestamp: ${timestamp}`);
          return '';
        }

        if (days >= 7) {
          // 7+ days: show date
          return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        } else if (hours >= 24) {
          // 1-7 days: show date and time
          return d.toLocaleString('en-US', { 
            month: 'short', 
            day: 'numeric', 
            hour: 'numeric', 
            minute: '2-digit' 
          });
        } else if (hours >= 6) {
          // 6-24 hours: show time with date if crossing midnight
          // FIXED: Use index parameter instead of indexOf to avoid bugs
          const prevVal = index > 0 ? splits[index - 1] : null;
          if (prevVal !== null) {
            const prevDate = new Date(prevVal);
            // Check if we crossed midnight (different day)
            if (!isNaN(prevDate.getTime()) && d.getDate() !== prevDate.getDate()) {
              return d.toLocaleString('en-US', { 
                month: 'short', 
                day: 'numeric', 
                hour: 'numeric', 
                minute: '2-digit' 
              });
            }
          }
          return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
        } else {
          // < 6 hours: show time
          return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
        }
      });
    };

    // Chart options
    const opts: uPlotTypes.Options = {
      title: 'Weather Readings',
      width: chartArea.offsetWidth || container.offsetWidth || 800,
      height: 400,
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
          range: [-60, 60],
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
          range: [0, 5],
        },
      },
      series: [
        {},
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
          grid: { show: true, stroke: '#607D8B', width: 1 },
          time: true,
          values: (u, splits) => {
            // Format time labels based on the data range
            // Use scale if available, otherwise fallback to data range
            const min = (u.scales.x?.min ?? times[0]) as number;
            const max = (u.scales.x?.max ?? times[times.length - 1]) as number;
            return formatTimeLabels(splits, min, max);
          },
        },
        {
          scale: 'y',
          stroke: '#607D8B',
          grid: { show: showYAxes, stroke: '#607D8B', width: 1 },
          show: showYAxes,
          label: 'Temperature (°C)',
        },
        {
          scale: 'y1',
          stroke: '#607D8B',
          grid: { show: false },
          show: showYAxes,
          label: 'Humidity (%)',
        },
        {
          scale: 'y2',
          stroke: '#607D8B',
          grid: { show: false },
          show: showYAxes,
          label: 'Pressure (hPa)',
        },
        {
          scale: 'y3',
          stroke: '#607D8B',
          grid: { show: false },
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
    };

    // Create chart
    try {
      chart = new uPlotLib(opts, data, chartArea);
      // Add toggle controls if they don't exist
      addToggleControls(container);
    } catch (e) {
      console.error('Failed to create chart:', e);
      chartArea.innerHTML = '<p class="no-data">Failed to render chart</p>';
    }
  }

  function addToggleControls(container: HTMLElement): void {
    // Chart container is now in the partial, so controls are already set up by htmx
    // Just update showYAxes from server state
    showYAxes = getInitialShowYAxes();
    
    // Set up htmx event listener for the toggle button
    // Note: Button is recreated on each swap, so we always need to attach the listener
    const axesToggle = document.getElementById('axes-toggle');
    if (axesToggle) {
      // Remove any existing handler to avoid duplicates
      const existingHandler = (axesToggle as any)._axesToggleHandler;
      if (existingHandler) {
        axesToggle.removeEventListener('htmx:configRequest', existingHandler);
      }
      
      // Create and attach new handler
      const handler = function(event: Event) {
        const customEvent = event as CustomEvent;
        // Get current state and toggle it
        const currentState = axesToggle.dataset.showYAxes === 'true';
        const newState = !currentState;
        
        // Add show_y_axes parameter to the request
        (customEvent.detail as any).parameters = (customEvent.detail as any).parameters || {};
        (customEvent.detail as any).parameters.show_y_axes = String(newState);
      };
      (axesToggle as any)._axesToggleHandler = handler;
      axesToggle.addEventListener('htmx:configRequest', handler);
    }
  }

  // Listen for swaps - htmx handles everything
  // Use afterSettle to catch all swaps including out-of-band
  function reinitChart(): void {
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
  let resizeTimeout: ReturnType<typeof setTimeout>;
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
          axesToggle.setAttribute('data-show-y-axes', showYAxes.toString());
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
