import { Chart, ChartConfiguration } from "chart.js/auto";
import zoomPlugin from "chartjs-plugin-zoom";
import "chartjs-adapter-date-fns";
import { ChartButtons, DataRow, Metadata } from "./types";
import { cloneDeep, merge } from "lodash";
import classes from "./styles/dynamic-styles.module.css";
import type { ZoomPluginOptions } from "chartjs-plugin-zoom/types/options";

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
    },
  },
};

export class WesplotChart {
  private _config: ChartConfiguration<"scatter"> | undefined;
  private _canvas: HTMLCanvasElement;
  private _metadata: Metadata;
  private _chart: Chart;
  private _buttons: ChartButtons;
  private _title: Element;
  private _zoom_active: boolean;
  private _pan_active: boolean;
  private _zoom_plugin_options: ZoomPluginOptions = {
    zoom: {
      wheel: {
        enabled: false,
      },
      pinch: {
        enabled: false,
      },
      drag: {
        enabled: false,
      },
      mode: "x",
    },
    pan: {
      enabled: true, // This must be initialized to true, or it won't work
      mode: "xy",
    },
  };

  constructor(panel: HTMLElement, metadata: Metadata) {
    // ==========================
    // Get relevant HTML elements
    // ==========================

    this._canvas = panel.getElementsByTagName(
      "canvas"
    )[0]! as HTMLCanvasElement;
    this._title = panel.getElementsByClassName("title-text")[0]!;
    this._buttons = {
      screenshot: panel.getElementsByClassName(
        "screenshot"
      )[0]! as HTMLButtonElement,

      resetzoom: panel.getElementsByClassName(
        "reset-zoom"
      )[0]! as HTMLButtonElement,

      zoom: panel.getElementsByClassName("zoom")[0]! as HTMLButtonElement,

      pan: panel.getElementsByClassName("pan")[0]! as HTMLButtonElement,

      settings: panel.getElementsByClassName(
        "settings"
      )[0]! as HTMLButtonElement,
    };

    this._buttons.screenshot.addEventListener(
      "click",
      this.screenshot.bind(this)
    );
    this._buttons.resetzoom.addEventListener(
      "click",
      this.resetView.bind(this)
    );
    this._buttons.zoom.addEventListener("click", this.toggleZoom.bind(this));
    this._buttons.pan.addEventListener("click", this.togglePan.bind(this));
    this._buttons.settings.addEventListener("click", this.settings.bind(this));

    if (metadata.ChartOptions.Title) {
      this.setTitle(metadata.ChartOptions.Title);
    }

    // Zoom and pan are not enabled by default
    this._zoom_active = false;
    this._pan_active = false;

    // =======================
    // Set chart configuration
    // =======================

    this._metadata = metadata;
    this._config = cloneDeep(default_config); // Deep copy

    if (!this._metadata.XIsTimestamp) {
      this._config.options!.scales!.x!.type = "linear";
    }

    // We need to maintain a stable reference to zoom plugin options so it can
    // be accessed and mutated in the zoom/pan button handlers.
    this._config.options!.plugins!.zoom = this._zoom_plugin_options;

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
    this._chart = new Chart(this._canvas, this._config);

    // TODO: Possible upstream bug, cannot pan if pan is not initially enabled?
    // For now, set pan enabled by default on and immediately disable it...
    this.setZoomPan("pan", false);
  }

  setTitle(title: string) {
    this._title.textContent = title;
  }

  update(rows: DataRow[]) {
    for (const [i, _] of this._metadata.Columns.entries()) {
      const data = this._chart.data.datasets[i].data;
      for (const row of rows) {
        data.push({ x: row.X, y: row.Ys[i] });
        this._chart.data.datasets[i].data = data;
        if (data.length > this._metadata.WindowSize) {
          data.shift();
        }
      }
    }
    // Do not animate
    this._chart.update("none");
  }

  private addUnits(value: string | number, _index: unknown, _ticks: unknown) {
    if (!this._metadata.YUnit) {
      return value; // Don't append space if no unit is provided
    }

    if (typeof value === "number") {
      return `${value.toFixed(3)} ${this._metadata.YUnit}`; // TODO: fix this
    }

    return `${value} ${!this._metadata.YUnit}`;
  }

  private screenshot(_event: unknown) {
    // Set canvas background color to white for the screenshot
    const context = this._canvas.getContext("2d")!;
    context.save();
    context.globalCompositeOperation = "destination-over";
    context.fillStyle = "white";
    context.fillRect(0, 0, this._canvas.width, this._canvas.height);
    context.restore(); // This will paint the background white until the next chart update

    var a = document.createElement("a");
    // a.href = this._chart.toBase64Image();
    a.href = this._canvas.toDataURL("image/png", 1.0);
    a.download = `wesplot_${this._metadata.ChartOptions.Title}.png`;

    // Trigger the download
    a.click();

    this._chart.update("none"); // Update the chart to reset the background color
  }

  private resetView(_event: unknown) {
    this._chart.resetZoom();
  }

  private toggleZoom(_event: unknown) {
    this.setZoomPan("zoom", !this._zoom_active);
  }

  private togglePan(_event: unknown) {
    this.setZoomPan("pan", !this._pan_active);
  }

  private setZoomPan(type: "zoom" | "pan", value: boolean) {
    if (type === "zoom") {
      this._zoom_active = value;
      if (value && this._pan_active) {
        this._pan_active = false;
      }
    } else {
      this._pan_active = value;
      if (value && this._zoom_active) {
        this._zoom_active = false;
      }
    }

    if (this._zoom_active) {
      this._buttons.zoom.classList.add(classes["button-on"]);
    } else {
      this._buttons.zoom.classList.remove(classes["button-on"]);
    }

    if (this._pan_active) {
      this._buttons.pan.classList.add(classes["button-on"]);
    } else {
      this._buttons.pan.classList.remove(classes["button-on"]);
    }

    this._zoom_plugin_options.zoom!.drag!.enabled = this._zoom_active;
    this._zoom_plugin_options.zoom!.pinch!.enabled = this._zoom_active;
    this._zoom_plugin_options.zoom!.wheel!.enabled = this._zoom_active;

    this._zoom_plugin_options.pan!.enabled = this._pan_active;
    console.log(this._chart.config);
    this._chart.update("none");
  }

  private settings(_event: unknown) {}
}
