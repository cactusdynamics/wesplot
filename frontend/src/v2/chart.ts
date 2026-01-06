import {
  CategoryScale,
  type ChartConfiguration,
  Chart as ChartJS,
  Legend,
  LinearScale,
  LineElement,
  type Point,
  PointElement,
  TimeScale,
  Title,
  Tooltip,
} from "chart.js";
import "chartjs-adapter-date-fns";
import zoomPlugin from "chartjs-plugin-zoom";
import type { ZoomPluginOptions } from "chartjs-plugin-zoom/types/options";

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  TimeScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  zoomPlugin,
);

// Monotonic counter used to give each Chart instance a unique id for
// performance marks/measures.
let _nextChartId = 0;

export interface ChartOptions {
  title?: string;
  xLabel?: string;
  yLabel?: string;
  xMin?: number;
  xMax?: number;
  yMin?: number;
  yMax?: number;
  columns?: string[];
  xIsTimestamp?: boolean;
}

export interface ChartConfig {
  container: HTMLElement; // DOM element to render chart into
  seriesIds: number[]; // Which series to display (series identifiers)
  options: ChartOptions; // Chart options
  colors?: string[]; // Series colors
}

interface SeriesState {
  xSegments: Float64Array[];
  ySegments: Float64Array[];
  generation: number; // Incremented on each update
  lastRenderedGeneration: number; // Last generation rendered
  datasetIndex: number; // Index in Chart.js datasets array
}

// Interaction mode: mutually exclusive states for user interaction
export enum InteractionMode {
  None = "none",
  Zoom = "zoom",
  Pan = "pan",
}

export class Chart {
  private _chart: ChartJS;
  private _chartId: number;
  private _series: Map<number, SeriesState> = new Map();
  private _options: ChartOptions;
  private _renderScheduled = false;
  private _rafId: number | null = null;

  // Stable zoom plugin options so callers can programmatically toggle zoom/pan.
  private _zoomPluginOptions: ZoomPluginOptions = {
    pan: {
      enabled: false,
      mode: "xy",
    },
    zoom: {
      wheel: { enabled: false },
      pinch: { enabled: false },
      drag: { enabled: false },
      mode: "x",
    },
  };

  // Track active mode (mutually exclusive).
  private _activeMode: InteractionMode = InteractionMode.None;

  constructor(config: ChartConfig) {
    this._options = config.options;
    // Assign a unique id for this chart instance for performance spans
    this._chartId = ++_nextChartId;

    // Create Chart.js configuration
    const chartConfig: ChartConfiguration = {
      type: "line",
      data: {
        datasets: [],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        scales: {
          x: {
            type: this._options.xIsTimestamp ? "time" : "linear",
            title: {
              display: true,
              text: this._options.xLabel || "X",
            },
            min: this._options.xMin,
            max: this._options.xMax,
          },
          y: {
            title: {
              display: true,
              text: this._options.yLabel || "Y",
            },
            min: this._options.yMin,
            max: this._options.yMax,
          },
        },
        plugins: {
          title: {
            display: true,
            text: this._options.title || "Wesplot",
          },
          legend: {
            display: true,
            position: "bottom",
          },
          zoom: this._zoomPluginOptions,
        },
      },
    };

    // Create Chart.js instance
    const canvas = document.createElement("canvas");
    config.container.appendChild(canvas);
    this._chart = new ChartJS(canvas, chartConfig);

    // Initialize datasets for configured series
    for (const seriesId of config.seriesIds) {
      this._getOrCreateSeries(seriesId, config.colors);
    }
  }

  /**
   * Update chart with new data (allocation-free, fast).
   * Stores segment references and schedules a render if needed.
   */
  update(
    seriesId: number,
    xSegments: Float64Array[],
    ySegments: Float64Array[],
  ): void {
    // Validate input
    if (xSegments.length !== ySegments.length) {
      console.error(
        `Chart.update: xSegments and ySegments length mismatch for series ${seriesId}`,
      );
      return;
    }

    const series = this._getOrCreateSeries(seriesId);

    // Store references (no copy) and bump generation
    series.xSegments = xSegments;
    series.ySegments = ySegments;
    series.generation++;

    // Schedule render if not already scheduled
    if (!this._renderScheduled) {
      this._renderScheduled = true;
      this._rafId = requestAnimationFrame(this._renderFrame);
    }
  }

  /**
   * Render frame callback (called by requestAnimationFrame).
   * Converts segments to Chart.js data and updates the chart.
   */
  private _renderFrame = (_timestamp: number): void => {
    const start = performance.now();

    // Clear scheduling state
    this._renderScheduled = false;
    this._rafId = null;

    let changed = false;

    // Process each series
    for (const series of this._series.values()) {
      // Skip if already rendered
      if (series.generation === series.lastRenderedGeneration) {
        continue;
      }

      // Convert segments to Chart.js data
      const data = this._convertSegmentsToData(
        series.xSegments,
        series.ySegments,
      );

      // Update dataset
      this._chart.data.datasets[series.datasetIndex].data = data;
      series.lastRenderedGeneration = series.generation;
      changed = true;
    }

    // Update Chart.js if any dataset changed
    if (changed) {
      this._chart.update("none"); // 'none' mode = no animation
      const end = performance.now();
      performance.measure(`render-${this._chartId}`, { start, end });
    }
  };

  // Convert Float64Array segments to Chart.js data format.
  // Handles NaN sentinel values (discontinuities) by converting to null.
  private _convertSegmentsToData(
    xSegments: Float64Array[],
    ySegments: Float64Array[],
  ): (Point | null)[] {
    const data: (Point | null)[] = [];

    // Process each segment pair
    for (let segIdx = 0; segIdx < xSegments.length; segIdx++) {
      const xSeg = xSegments[segIdx];
      const ySeg = ySegments[segIdx];

      for (let i = 0; i < xSeg.length; i++) {
        const x = xSeg[i];
        const y = ySeg[i];

        // Convert NaN (discontinuity sentinel) to null for Chart.js
        if (Number.isNaN(x) || Number.isNaN(y)) {
          data.push(null);
        } else {
          data.push({ x, y });
        }
      }
    }

    return data;
  }

  /**
   * Get or create series state.
   * Creates a new Chart.js dataset if series doesn't exist.
   */
  private _getOrCreateSeries(seriesId: number, colors?: string[]): SeriesState {
    let series = this._series.get(seriesId);
    if (!series) {
      // Create new dataset for this series
      const datasetIndex = this._chart.data.datasets.length;
      const label = this._options.columns?.[seriesId] || `Series ${seriesId}`;
      const color = colors?.[seriesId] || this._getDefaultColor(seriesId);

      this._chart.data.datasets.push({
        label,
        data: [],
        borderColor: color,
        backgroundColor: color,
        pointRadius: 0, // No points for performance
        borderWidth: 1,
        spanGaps: false, // Don't connect across null values (discontinuities)
      });

      series = {
        xSegments: [],
        ySegments: [],
        generation: 0,
        lastRenderedGeneration: -1,
        datasetIndex,
      };
      this._series.set(seriesId, series);
    }
    return series;
  }

  /**
   * Get default color for a series.
   */
  private _getDefaultColor(seriesId: number): string {
    // Extended palette: 20+ visually distinct, colorblind-friendly colors
    const colors = [
      "#3366CC", // blue
      "#DC3912", // red
      "#FF9900", // orange
      "#109618", // green
      "#990099", // purple
      "#3B3EAC", // indigo
      "#0099C6", // cyan
      "#DD4477", // pink
      "#66AA00", // lime
      "#B82E2E", // dark red
      "#316395", // steel blue
      "#994499", // violet
      "#22AA99", // teal
      "#AAAA11", // olive
      "#6633CC", // deep purple
      "#E67300", // dark orange
      "#8B0707", // maroon
      "#329262", // dark green
      "#5574A6", // muted blue
      "#3B3EAC", // indigo (repeat for wraparound)
      "#B77322", // brown
      "#16D620", // bright green
      "#B91383", // magenta
      "#F4359E", // hot pink
      "#9C5935", // sienna
      "#A9C413", // yellow-green
      "#2A778D", // blue-green
      "#668D1C", // olive green
      "#BEA413", // gold
      "#0C5922", // forest green
    ];
    return colors[seriesId % colors.length];
  }

  /**
   * Update chart options.
   */
  updateOptions(options: Partial<ChartOptions>): void {
    if (options.title !== undefined && this._chart.options.plugins?.title) {
      this._chart.options.plugins.title.text = options.title;
    }
    if (options.xLabel !== undefined && this._chart.options.scales?.x) {
      // @ts-expect-error - Chart.js types are complex; title exists at runtime
      this._chart.options.scales.x.title = {
        display: true,
        text: options.xLabel,
      };
    }
    if (options.yLabel !== undefined && this._chart.options.scales?.y) {
      // @ts-expect-error - Chart.js types are complex; title exists at runtime
      this._chart.options.scales.y.title = {
        display: true,
        text: options.yLabel,
      };
    }
    this._chart.update("none");
  }

  /**
   * Get or set the active interaction mode. Use `InteractionMode.None` to clear.
   */
  get zoomPanMode(): InteractionMode {
    return this._activeMode;
  }

  set zoomPanMode(mode: InteractionMode) {
    // Set booleans based on the requested mode
    this._activeMode = mode;

    const isZoom = this._activeMode === InteractionMode.Zoom;
    const isPan = this._activeMode === InteractionMode.Pan;

    const zoom = this._zoomPluginOptions.zoom;
    if (zoom) {
      if (zoom.drag) {
        zoom.drag.enabled = isZoom;
      }
      if (zoom.pinch) {
        zoom.pinch.enabled = isZoom;
      }
      if (zoom.wheel) {
        zoom.wheel.enabled = isZoom;
      }
    }

    const pan = this._zoomPluginOptions.pan;
    if (pan) {
      pan.enabled = isPan;
    }

    // Apply change to Chart.js
    this._chart.update("none");
  }

  /**
   * Destroy chart and clean up resources.
   */
  destroy(): void {
    // Cancel pending RAF
    if (this._rafId !== null) {
      cancelAnimationFrame(this._rafId);
      this._rafId = null;
      this._renderScheduled = false;
    }

    // Destroy Chart.js instance
    this._chart.destroy();

    // Clear series state
    this._series.clear();
  }
}
