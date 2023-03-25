import { Chart, ChartConfiguration } from "chart.js/auto";
import zoomPlugin from "chartjs-plugin-zoom";
import "chartjs-adapter-date-fns";
import { DataRow, Metadata } from "./types";
import { cloneDeep, merge } from "lodash";

Chart.defaults.font.size = 16;
Chart.register(zoomPlugin);

const default_config: ChartConfiguration<"scatter"> = {
  type: "scatter",
  data: {
    datasets: [],
  },
  options: {
    showLine: true,
    maintainAspectRatio: false,
    scales: {
      x: {
        title: {
          display: true,
        },
        type: "time",
      },
      y: {
        beginAtZero: true,
        title: {
          display: true,
        },
        ticks: {
          precision: 3,
        },
      },
    },
    plugins: {
      title: {
        display: true,
        font: {
          size: 20,
        },
      },
      tooltip: {
        enabled: true,
      },
      legend: {
        position: "bottom",
      },
      zoom: {
        zoom: {
          wheel: {
            enabled: true,
          },
          pinch: {
            enabled: true,
          },
          mode: "xy",
        },
        pan: {
          enabled: true,
          mode: "xy",
        },
      },
    },
  },
};

export class WesplotChart {
  ctx: HTMLCanvasElement;
  private _config: ChartConfiguration<"scatter"> | undefined;
  private _metadata: Metadata;
  private _chart: Chart;

  // Normal signature with defaults
  constructor(ctx: HTMLCanvasElement, metadata: Metadata) {
    this.ctx = ctx;
    this._metadata = metadata;
    this._config = cloneDeep(default_config); // Deep copy

    // Merge in config parameters from metadata
    merge(this._config, {
      options: {
        scales: {
          x: {
            title: { text: metadata.ChartOptions.XLabel },
          },
          y: {
            title: { text: metadata.ChartOptions.YLabel },
            ticks: { callback: this.addUnits.bind(this) },
          },
        },
        plugins: {
          title: { text: metadata.ChartOptions.Title },
        },
      },
    });

    // Initialize a dataset for each data column as specified by the metadata
    for (const column of metadata.Columns) {
      this._config.data.datasets.push({
        label: column,
        data: [],
        borderWidth: 1,
      });
    }

    // Create the chart
    this._chart = new Chart(ctx, this._config);
  }

  addUnits(value: number, _index: unknown, _ticks: unknown) {
    if (!this._metadata.YUnit) {
      return value; // Don't append space if no unit is provided
    }
    if (typeof value === "number") {
      return `${value.toFixed(3)} ${!this._metadata.YUnit}`; // TODO: fix this
    }
    return `${value} ${!this._metadata.YUnit}`;
  }

  get Config() {
    return this._config!;
  }
  set Config(config: ChartConfiguration<"scatter">) {
    this._config = config;
  }

  update(rows: DataRow[]) {
    for (const [i, _] of this._metadata.Columns.entries()) {
      const data = this._chart.data.datasets[i].data;
      for (const row of rows) {
        data.push({ x: row.Timestamp, y: row.Data[i] });
        this._chart.data.datasets[i].data = data;
        if (data.length > this._metadata.WindowSize) {
          data.shift();
        }
      }
    }

    // Do not animate
    this._chart.update("none");
  }
}
