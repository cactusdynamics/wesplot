/**
 * Wesplot v2 Main Application
 *
 * Connects Streamer to Chart and manages UI state.
 */

import { Chart, type ChartOptions } from "./chart.js";
import { Streamer } from "./streamer.js";
import type { Metadata } from "./types.js";

// Import styles (CSS imports don't use .js extension in Vite)
// biome-ignore lint/correctness/useImportExtensions: CSS imports are handled by Vite bundler
import "../styles/app.css";

let baseHost = location.host;
if (import.meta.env.DEV) {
  // This does mean in development, we can only run one of these at a time,
  // which I think is fine.
  baseHost = `${location.hostname}:5274`;
}

// DOM elements - retrieve on demand to ensure DOM is loaded
function getElement<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (!element) throw new Error(`${id} element not found`);
  return element as T;
}

/**
 * Handles the status bar at the bottom of the page.
 */
class StatusBar {
  private statusText: HTMLElement;
  private liveIndicator: HTMLElement;
  private notLiveIndicator: HTMLElement;
  private errorIndicator: HTMLElement;
  private pauseButton: HTMLButtonElement;
  private chartWrapper: ChartWrapper;

  constructor(chartWrapper: ChartWrapper) {
    this.chartWrapper = chartWrapper;
    this.statusText = getElement<HTMLElement>("status-text");
    this.liveIndicator = getElement<HTMLElement>("live-indicator");
    this.notLiveIndicator = getElement<HTMLElement>("not-live-indicator");
    this.errorIndicator = getElement<HTMLElement>("error-indicator");
    this.pauseButton = getElement<HTMLButtonElement>("btn-pause");
    this.setupPauseButton();
  }

  updateStatus(
    message: string,
    state: "connecting" | "live" | "paused" | "error" | "disconnected",
  ): void {
    this.statusText.textContent = message;

    // Update indicators
    this.liveIndicator.style.display =
      state === "live" ? "inline-block" : "none";
    this.notLiveIndicator.style.display =
      state === "connecting" || state === "paused" || state === "disconnected"
        ? "inline-block"
        : "none";
    this.errorIndicator.style.display =
      state === "error" ? "inline-block" : "none";
  }

  private setupPauseButton(): void {
    this.pauseButton.addEventListener("click", () => {
      const isPaused = !this.chartWrapper.isPaused;
      this.chartWrapper.setPaused(isPaused);

      const icon = this.pauseButton.querySelector("i");
      if (!icon) return;

      if (isPaused) {
        icon.className = "fa-solid fa-play";
        icon.title = "Resume";
        this.updateStatus("Paused", "paused");
      } else {
        icon.className = "fa-solid fa-pause";
        icon.title = "Pause";
        this.updateStatus("Streaming", "live");
      }
    });
  }
}

/**
 * Handles the chart and its title bar with action buttons.
 */
class ChartWrapper {
  private panelContainer: HTMLElement;
  private titleBar: HTMLElement | null = null;
  private titleText: HTMLElement | null = null;
  private chartArea: HTMLElement | null = null;
  private chartContainer: HTMLElement | null = null;
  private chart: Chart | null = null;
  private _isPaused = false;
  private screenshotBtn: HTMLButtonElement | null = null;
  private resetZoomBtn: HTMLButtonElement | null = null;
  private zoomBtn: HTMLButtonElement | null = null;
  private panBtn: HTMLButtonElement | null = null;
  private settingsBtn: HTMLButtonElement | null = null;

  constructor() {
    this.panelContainer = getElement<HTMLElement>("panel");
  }

  setPaused(paused: boolean): void {
    this._isPaused = paused;
  }

  get isPaused(): boolean {
    return this._isPaused;
  }

  createChart(metadata: Metadata): void {
    console.log("Metadata received:", metadata);

    // Create title bar
    this.createTitleBar(metadata.WesplotOptions.Title || "Wesplot v2");

    // Create chart area
    this.chartArea = document.createElement("div");
    this.chartArea.className = "chart-area";
    this.panelContainer.appendChild(this.chartArea);

    // Create chart container
    this.chartContainer = document.createElement("div");
    this.chartContainer.className = "chartjs-container";
    this.chartArea.appendChild(this.chartContainer);

    // Create chart options from metadata
    const chartOptions: ChartOptions = {
      title: metadata.WesplotOptions.Title,
      showTitle: false,
      xLabel: metadata.WesplotOptions.XLabel,
      yLabel: metadata.WesplotOptions.YLabel,
      xMin: metadata.WesplotOptions.XMin,
      xMax: metadata.WesplotOptions.XMax,
      yMin: metadata.WesplotOptions.YMin,
      yMax: metadata.WesplotOptions.YMax,
      yUnit: metadata.WesplotOptions.YUnit,
      columns: metadata.WesplotOptions.Columns,
      xIsTimestamp: metadata.XIsTimestamp,
    };

    // Determine which series to display (all of them for now)
    const numSeries = metadata.WesplotOptions.Columns.length;
    const seriesIds = Array.from({ length: numSeries }, (_, i) => i);

    // Create chart
    this.chart = new Chart({
      container: this.chartContainer,
      seriesIds,
      options: chartOptions,
    });
  }

  private createTitleBar(title: string): void {
    // Create title bar
    this.titleBar = document.createElement("div");
    this.titleBar.className = "title-bar";

    // Create title text
    this.titleText = document.createElement("div");
    this.titleText.className = "title-text";
    this.titleText.textContent = title;
    this.titleBar.appendChild(this.titleText);

    // Create button bar
    const buttonBar = document.createElement("div");
    buttonBar.className = "button-bar";

    // Create buttons
    this.screenshotBtn = this.createButton("fa-camera", "Save image");
    this.resetZoomBtn = this.createButton("fa-expand", "Reset zoom");
    this.zoomBtn = this.createButton("fa-magnifying-glass", "Zoom");
    this.panBtn = this.createButton("fa-arrows-up-down-left-right", "Pan");
    this.settingsBtn = this.createButton("fa-gear", "Settings");

    buttonBar.appendChild(this.screenshotBtn);
    buttonBar.appendChild(this.resetZoomBtn);
    buttonBar.appendChild(this.zoomBtn);
    buttonBar.appendChild(this.panBtn);
    buttonBar.appendChild(this.settingsBtn);

    this.titleBar.appendChild(buttonBar);
    this.panelContainer.appendChild(this.titleBar);

    // Setup button event listeners
    this.setupButtons();
  }

  private createButton(iconClass: string, title: string): HTMLButtonElement {
    const button = document.createElement("button");
    const icon = document.createElement("i");
    icon.className = `fa-solid ${iconClass}`;
    icon.title = title;
    button.appendChild(icon);
    return button;
  }

  updateData(
    seriesId: number,
    xSegments: Float64Array[],
    ySegments: Float64Array[],
  ): void {
    if (!this.chart || this._isPaused) {
      return;
    }

    // Update chart with new data
    this.chart.update(seriesId, xSegments, ySegments);
  }

  private setupButtons(): void {
    // Placeholder for button event listeners
    // Implement functionality as needed
    if (
      !this.screenshotBtn ||
      !this.resetZoomBtn ||
      !this.zoomBtn ||
      !this.panBtn ||
      !this.settingsBtn
    ) {
      console.error("Buttons not initialized");
      return;
    }

    this.screenshotBtn.addEventListener("click", () => {
      console.log("Screenshot button clicked");
    });
    this.resetZoomBtn.addEventListener("click", () => {
      console.log("Reset zoom button clicked");
    });
    this.zoomBtn.addEventListener("click", () => {
      console.log("Zoom button clicked");
    });
    this.panBtn.addEventListener("click", () => {
      console.log("Pan button clicked");
    });
    this.settingsBtn.addEventListener("click", () => {
      console.log("Settings button clicked");
    });
  }
}

// State
let streamer: Streamer | null = null;
let statusBar: StatusBar | null = null;
let chartWrapper: ChartWrapper | null = null;

// Initialize application
async function main() {
  try {
    chartWrapper = new ChartWrapper();
    statusBar = new StatusBar(chartWrapper);

    statusBar.updateStatus("Connecting...", "connecting");

    // Create streamer with 1000 point window size
    streamer = new Streamer(`ws://${baseHost}/ws2`, 1000);

    // Register callbacks
    streamer.registerCallbacks({
      onMetadata: handleMetadata,
      onData: handleData,
      onStreamEnd: handleStreamEnd,
      onError: handleError,
    });

    // Connect to WebSocket
    await streamer.connect();
    statusBar.updateStatus("Connected", "live");
  } catch (error) {
    handleError(error instanceof Error ? error : new Error(String(error)));
  }
}

function handleMetadata(metadata: Metadata): void {
  if (!statusBar || !chartWrapper) return;

  // Create chart with title bar
  chartWrapper.createChart(metadata);

  statusBar.updateStatus("Streaming", "live");
}

function handleData(
  seriesId: number,
  xSegments: Float64Array[],
  ySegments: Float64Array[],
): void {
  if (!chartWrapper) return;
  chartWrapper.updateData(seriesId, xSegments, ySegments);
}

function handleStreamEnd(error: boolean, message: string): void {
  if (!statusBar) return;
  const statusMessage = error
    ? `Error: ${message}`
    : `Stream ended: ${message}`;
  statusBar.updateStatus(statusMessage, error ? "error" : "disconnected");
  console.log(`Stream ended: ${message}`);
}

function handleError(error: Error): void {
  if (!statusBar) return;
  statusBar.updateStatus(`Error: ${error.message}`, "error");
  console.error("Streamer error:", error);
}

// Start application when DOM is ready
window.addEventListener("load", () => {
  main();
});
