import { Chart, ChartConfiguration } from "chart.js/auto";
import zoomPlugin from "chartjs-plugin-zoom";
import "chartjs-adapter-date-fns";
import {
  ChartButtons,
  DataRow,
  Metadata,
  SettingsPanelInputs,
  WesplotOptions,
} from "./types";
import { cloneDeep, merge } from "lodash-es";
import classes from "./styles/dynamic-styles.module.css";
import type { ZoomPluginOptions } from "chartjs-plugin-zoom/types/options";
import { LimitInput } from "./limits";

Chart.defaults.font.size = 16;
Chart.register(zoomPlugin);
Chart.defaults.elements.point.borderWidth = 0;
Chart.defaults.elements.point.radius = 1;
// Chart.defaults.elements.point.

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
        title: {
          display: true,
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
  // HTML elements
  private _config: ChartConfiguration<"scatter"> | undefined;
  private _canvas: HTMLCanvasElement;
  private _title: Element;
  private _settings_panel: HTMLDivElement;
  private _settings_status_bar: HTMLDivElement;
  private _settings_status_text: HTMLDivElement;

  private _metadata: Metadata; // Contains the metadata passed from the command line
  private _chart: Chart; // The chart object itself
  private _buttons: ChartButtons; // A container for the buttons in the top right
  private _settings: SettingsPanelInputs;
  private _x0: number = NaN; // To zero the X-axis

  private _wesplot_options: WesplotOptions;

  // States
  private _zoom_active: boolean;
  private _pan_active: boolean;

  // Config for the zoom plugin
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

    // Settings panel
    this._settings_panel = document.getElementById(
      "settings-panel"
    )! as HTMLDivElement;
    this._settings_status_bar = document.getElementsByClassName(
      "status-bar"
    )![0] as HTMLDivElement;
    this._settings_status_text = document.getElementById(
      "settings-status-text"
    )! as HTMLParagraphElement;

    // Inputs within settings panel
    this._settings = {
      cancel: document.getElementById(
        "settings-cancel-button"
      )! as HTMLButtonElement,
      save: document.getElementById(
        "settings-save-button"
      )! as HTMLButtonElement,
      title: document.getElementById("settings-title")! as HTMLInputElement,
      series_names: document.getElementById(
        "settings-series-names"
      )! as HTMLInputElement,
      x_min: new LimitInput(
        document.getElementById("settings-xmin")! as HTMLInputElement,
        metadata.XIsTimestamp
      ),
      x_max: new LimitInput(
        document.getElementById("settings-xmax")! as HTMLInputElement,
        metadata.XIsTimestamp
      ),
      x_label: document.getElementById("settings-xlabel")! as HTMLInputElement,
      y_min: new LimitInput(
        document.getElementById("settings-ymin")! as HTMLInputElement
      ),
      y_max: new LimitInput(
        document.getElementById("settings-ymax")! as HTMLInputElement
      ),
      y_label: document.getElementById("settings-ylabel")! as HTMLInputElement,
      y_unit: document.getElementById("settings-yunit")! as HTMLInputElement,
      relative_start: document.getElementById(
        "settings-relative-start"
      )! as HTMLInputElement,
    };

    // Event handlers to settings panel buttons
    this._settings.cancel.addEventListener(
      "click",
      this.closeSettings.bind(this)
    );
    this._settings.save.addEventListener("click", this.saveSettings.bind(this));

    // The canvas
    this._canvas = panel.getElementsByTagName(
      "canvas"
    )[0]! as HTMLCanvasElement;

    // The title
    this._title = panel.getElementsByClassName("title-text")[0]!;

    // The buttons on the top right
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

    // Event handlers for top right buttons
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
    this._buttons.settings.addEventListener(
      "click",
      this.openSettings.bind(this)
    );

    const self = this;
    // Event handler to close settings if clicking outside the panel
    document
      .getElementById("settings-panel")!
      .addEventListener("click", function (event) {
        if (event.target === this) {
          // This is now the "settings-panel" element
          // Capture self to be able to call functions from the WesplotChart object
          self.closeSettings();
        }
      });

    // Event handler to close settings on Esc
    document.addEventListener(
      "keydown",
      (event) => {
        if (event.defaultPrevented) {
          return; // Do nothing if the event was already processed
        }

        switch (event.key) {
          case "Esc": // IE/Edge specific value
          case "Escape":
            // Do something for "esc" key press.
            this.closeSettings();
            break;
          default:
            return; // Quit when this doesn't handle the key event.
        }

        // Cancel the default action to avoid it being handled twice
        event.preventDefault();
      },
      true
    );

    if (metadata.WesplotOptions.Title) {
      this.setTitle(metadata.WesplotOptions.Title);
    }

    // Zoom and pan are not enabled by default
    this._zoom_active = false;
    this._pan_active = false;

    // =======================
    // Set chart configuration
    // =======================

    this._metadata = metadata;
    this._config = cloneDeep(default_config); // Deep copy
    this._wesplot_options = metadata.WesplotOptions;

    // Set a linear timescape if we are not using timestamped data or if we have a relative start
    if (!this._metadata.XIsTimestamp || this._metadata.RelativeStart) {
      this._config.options!.scales!.x!.type = "linear";
    }

    // We need to maintain a stable reference to zoom plugin options so it can
    // be accessed and mutated in the zoom/pan button handlers.
    this._config.options!.plugins!.zoom = this._zoom_plugin_options;

    // Initialize a dataset for each data column as specified by the metadata
    for (const column of this._wesplot_options.Columns) {
      this._config.data.datasets.push({
        label: column,
        data: [],
        borderWidth: 1,
      });
    }

    // Do not display legend for 1 data set
    if (this._wesplot_options.Columns.length < 2) {
      this._config.options!.plugins!.legend!.display = false;
    }

    this.updatePlotSettings();
    // Create the chart
    this._chart = new Chart(this._canvas, this._config);

    // TODO: Possible upstream bug, cannot pan if pan is not initially enabled?
    // For now, set pan enabled by default on and immediately disable it...
    this.setZoomPan("pan", false);
  }

  updatePlotSettings() {
    this.setTitle(this._wesplot_options.Title);
    for (const [index, column] of this._wesplot_options.Columns.entries()) {
      this._config!.data.datasets[index].label = column;
    }
    merge(this._config, {
      options: {
        scales: {
          x: {
            title: { text: this._wesplot_options.XLabel },
            min: this._wesplot_options.XMin,
            max: this._wesplot_options.XMax,
          },
          y: {
            title: { text: this._wesplot_options.YLabel },
            ticks: { callback: this.addUnits.bind(this) },
            min: this._wesplot_options.YMin,
            max: this._wesplot_options.YMax,
          },
        },
      },
    });
    if (this._chart) {
      this._chart.update("none");
    }
  }

  setTitle(title: string) {
    this._title.textContent = title;
    document.title = title;
  }

  update(rows: DataRow[]) {
    for (const [i, _] of this._wesplot_options.Columns.entries()) {
      const data = this._chart.data.datasets[i].data;
      for (const row of rows) {
        let x = row.X;

        if (this._metadata.RelativeStart) {
          // Inefficient code, yay.
          // We want to display seconds if relative start is true, so we don't multiply
          if (Number.isNaN(this._x0)) {
            this._x0 = x;
          }

          x -= this._x0;
        } else if (this._metadata.XIsTimestamp) {
          // Server side seconds time in seconds.
          x *= 1000;
        }

        data.push([x, row.Ys[i]]);
        if (data.length > this._metadata.WindowSize) {
          data.shift();
        }
      }
    }

    // "none" means do not animate, this looks weird with an updating chart
    this._chart.update("none");
  }

  private addUnits(value: number | string, _index: unknown, _ticks: unknown) {
    if (typeof value === "number") {
      if (!this._wesplot_options.YUnit) {
        // Don't append space if no unit is provided
        return value.toPrecision(5).replace(/(?:\.0+|(\.\d+?)0+)$/, "$1");
      }
      return `${value.toPrecision(5).replace(/(?:\.0+|(\.\d+?)0+)$/, "$1")} ${
        this._wesplot_options.YUnit
      }`;
    }

    return `${value.replace(/(?:\.0+|(\.\d+?)0+)$/, "$1")} ${!this
      ._wesplot_options.YUnit}`;
  }

  private screenshot() {
    // Get the old title
    const cached_title = this._config!.options!.plugins!.title!.text;
    this._config!.options!.plugins!.title!.text = this._title.textContent!;
    this._chart.update("none");

    // Set canvas background color to white for the screenshot
    const context = this._canvas.getContext("2d")!;
    context.save();
    context.globalCompositeOperation = "destination-over";
    context.fillStyle = "white";
    context.fillRect(0, 0, this._canvas.width, this._canvas.height);
    context.restore(); // This will paint the background white until the next chart update

    var a = document.createElement("a");
    a.href = this._canvas.toDataURL("image/png", 1.0);
    a.download = `wesplot_${this._title.textContent!}.png`;

    // Trigger the download
    a.click();

    // Restore the old title
    this._config!.options!.plugins!.title!.text = cached_title;
    this._chart.update("none"); // Update the chart to reset the background color
  }

  private resetView() {
    // Must use NaN to reset limits back to auto
    this._wesplot_options.XMin = NaN;
    this._wesplot_options.XMax = NaN;
    this._wesplot_options.YMin = NaN;
    this._wesplot_options.YMax = NaN;
    this.updatePlotSettings();
    this._chart.resetZoom();
  }

  private toggleZoom() {
    this.setZoomPan("zoom", !this._zoom_active);
  }

  private togglePan() {
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
    this._chart.update("none");
  }

  private openSettings() {
    this.hideSettingsError();
    this._settings_panel.style.display = "flex";
    this._settings.title.value = this._wesplot_options.Title;
    this._settings.series_names.value = this._wesplot_options.Columns.join(",");

    // Display current X limits
    this._settings.x_min.set_value(this._wesplot_options.XMin);
    this._settings.x_max.set_value(this._wesplot_options.XMax);
    this._settings.x_label.value = this._wesplot_options.XLabel;

    // Display current Y limits
    this._settings.y_min.set_value(this._wesplot_options.YMin);
    this._settings.y_max.set_value(this._wesplot_options.YMax);

    this._settings.y_label.value = this._wesplot_options.YLabel;
    this._settings.y_unit.value = this._wesplot_options.YUnit;
  }

  private closeSettings() {
    this._settings_panel.style.display = "none";
  }

  private saveSettings(event: Event) {
    event.preventDefault(); // Disable automatic refresh on submit

    const num_columns = this._wesplot_options.Columns.length;
    const new_column_names = this._settings.series_names.value.split(",");
    const num_non_empty_new_colums = new_column_names
      .map((column_name): number => (column_name === "" ? 0 : 1))
      .reduce((partialSum, a) => partialSum + a, 0);

    if (new_column_names.length !== num_columns) {
      this.showSettingsError(
        `Error: Expected ${num_columns} column names, ${new_column_names.length} provided`
      );
      return;
    }

    if (new_column_names.length !== num_non_empty_new_colums) {
      this.showSettingsError(`Error: Some column names are empty`);
      return;
    }

    // Check that limits are valid:
    // XMax must be greater than XMin unless either are NaN
    if (
      !Number.isNaN(this._settings.x_min.get_value()) &&
      !Number.isNaN(this._settings.x_max.get_value()) &&
      this._settings.x_min.get_value() >= this._settings.x_max.get_value()
    ) {
      this.showSettingsError(`Error: X max must be greater than X min`);
      return;
    }
    if (
      !Number.isNaN(this._settings.y_min.get_value()) &&
      !Number.isNaN(this._settings.y_max.get_value()) &&
      this._settings.y_min.get_value() >= this._settings.y_max.get_value()
    ) {
      this.showSettingsError(`Error: Y max must be greater than Y min`);
      return;
    }
    this._wesplot_options.Title = this._settings.title.value;
    this._wesplot_options.Columns = new_column_names;

    this._wesplot_options.XMin = this._settings.x_min.get_value();
    this._wesplot_options.XMax = this._settings.x_max.get_value();

    this._wesplot_options.XLabel = this._settings.x_label.value;
    this._wesplot_options.YMin = this._settings.y_min.get_value();
    this._wesplot_options.YMax = this._settings.y_max.get_value();
    this._wesplot_options.YLabel = this._settings.y_label.value;
    this._wesplot_options.YUnit = this._settings.y_unit.value;

    this.updatePlotSettings();
    this.closeSettings();
  }

  showSettingsError(error_text: string) {
    this._settings_status_bar.style.display = "block";
    this._settings_status_text.textContent = error_text;
  }
  hideSettingsError() {
    this._settings_status_bar.style.display = "none";
    this._settings_status_text.textContent = "";
  }
}
