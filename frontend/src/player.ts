import { DataRow, StreamEndedMessage } from "./types";
import { WesplotChart } from "./wesplot-chart";

type PlayerState = "INIT" | "LIVE" | "ENDED" | "ERRORED";

export class Player {
  private _pause_button: HTMLButtonElement;
  private _pause_icon: HTMLElement;
  private _status_text: HTMLSpanElement;

  private _live_indicator: HTMLElement;
  private _not_live_indicator: HTMLElement;
  private _error_indicator: HTMLElement;

  private _paused: boolean = false;
  private _state: PlayerState = "INIT";
  private _error: string = "";

  private _chart?: WesplotChart;

  private _hostname: string;
  private _socket: WebSocket;
  private _data_buffer: DataRow[] = [];

  private _last_data_received_time?: number;
  private _interval_id: number;

  constructor(baseHost: string) {
    // Get all HTML elements
    // ---------------------

    // Pause button
    this._pause_button = document.getElementById(
      "btn-pause"
    )! as HTMLButtonElement;
    // Pause icon in pause button
    this._pause_icon = this._pause_button.getElementsByTagName("i")[0]!;
    this._pause_button.addEventListener("click", this.handlePause.bind(this));

    // Status text in bottom bar
    this._status_text = document.getElementById(
      "status-text"
    )! as HTMLSpanElement;

    // Live indicator
    this._live_indicator = document.getElementById("live-indicator")!;
    this._not_live_indicator = document.getElementById("not-live-indicator")!;
    this._error_indicator = document.getElementById("error-indicator")!;

    // Update the status text every second
    // -----------------------------------
    this._interval_id = setInterval(this.updateStatusBar.bind(this), 1000);

    // Set up websocket
    // ----------------

    this._hostname = `ws://${baseHost}/ws`;
    this._socket = new WebSocket(this._hostname);

    // Set socket handlers
    this._socket.addEventListener("open", () => {
      console.log("Successfully Connected");
      this._state = "LIVE";
      this.updateStatusBar();
    });

    // If the socket is closed, check for a StreamEndedMessage with the reason
    // why and update the status indicator and status text accordingly
    this._socket.addEventListener("close", async (event) => {
      console.log("Socket Closed Connection: ", event);
      // Clear the interval so status text is no longer updated
      clearInterval(this._interval_id);
      try {
        const response = await fetch(
          `${location.protocol}//${baseHost}/errors`
        );
        const error: StreamEndedMessage = await response.json();

        if (error.StreamError) {
          this.handleError(`Error (${error.StreamError})`);
        } else {
          // Stream ended without error
          this._state = "ENDED";
          this.updateStatusBar();
        }
      } catch (e) {
        this.handleError(`Stream aborted: ${e}`);
      }
    });

    this._socket.addEventListener("error", (error) => {
      this.handleError(`Websocket error: ${error}`);
    });

    // On receiving a message, parse the data and update the chart (or cache it if paused)
    this._socket.addEventListener("message", (event) => {
      this._last_data_received_time = Date.now();

      const rows: DataRow[] = JSON.parse(event.data);
      // If paused, append new data to the buffer, but do not push this to the chart
      // If not paused, no need to push to the buffer, update the chart directly
      if (this._paused) {
        this._data_buffer = this._data_buffer.concat(rows);
      } else {
        if (this._chart === undefined) {
          // TODO: There's a potential race here - this connection logic should happen after registerChart
          throw Error("Player has no registered chart");
        }

        this._chart.update(rows);
      }

      this.updateStatusBar();
    });
  }

  registerChart(chart: WesplotChart) {
    this._chart = chart;
  }

  handleError(error_text: string) {
    this._state = "ERRORED";
    this._error = error_text;
    this.updateStatusBar();
  }

  private handlePause(_event: MouseEvent) {
    this._paused = !this._paused;

    if (this._paused) {
      this._pause_icon.classList.add("fa-play");
      this._pause_icon.classList.remove("fa-pause");
      this._pause_icon.title = "Resume";
    } else {
      this._pause_icon.classList.add("fa-pause");
      this._pause_icon.classList.remove("fa-play");
      this._pause_icon.title = "Pause";

      // On resume, push the buffered data to the chart, and clear the buffer
      this._chart!.update(this._data_buffer);
      this._data_buffer = [];
    }

    this.updateStatusBar();
  }

  private updateStatusBar() {
    // If paused, override any other update to the status bar
    if (this._paused) {
      this.setIndicatorNotLive();
      this.setStatusText(`Paused: ${this._data_buffer.length} rows buffered`);
      return;
    }

    switch (this._state) {
      case "INIT":
        this.setIndicatorNotLive();
        this.setStatusText("Connecting...");
        break;
      case "LIVE":
        this.setIndicatorLive();
        if (this._last_data_received_time === undefined) {
          this.setStatusText("Live: no data received");
        } else {
          // Convert to seconds and round to nearest int to prevent noise
          const time_since_last_data = Math.round(
            (Date.now() - this._last_data_received_time) / 1000
          );
          this.setStatusText(
            `Live: last row received ${time_since_last_data} second(s) ago`
          );
        }
        break;
      case "ENDED":
        this.setIndicatorNotLive();
        this.setStatusText("Stream ended");
        break;
      case "ERRORED":
        this.setIndicatorError();
        this.setStatusText(this._error);
        break;
    }
  }

  private setIndicatorLive() {
    this._live_indicator.style.display = "inline-block";
    this._not_live_indicator.style.display = "none";
    this._error_indicator.style.display = "none";
  }

  private setIndicatorNotLive() {
    this._live_indicator.style.display = "none";
    this._not_live_indicator.style.display = "inline-block";
    this._error_indicator.style.display = "none";
  }

  private setIndicatorError() {
    this._live_indicator.style.display = "none";
    this._not_live_indicator.style.display = "none";
    this._error_indicator.style.display = "inline-block";
  }

  private setStatusText(status_text: string) {
    this._status_text.textContent = status_text;
  }
}
