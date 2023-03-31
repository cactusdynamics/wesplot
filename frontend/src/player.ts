import { DataRow, StreamEndedMessage } from "./types";
import { WesplotChart } from "./wesplot-chart";

export class Player {
  _pause_button: HTMLButtonElement;
  _pause_icon: HTMLElement;
  _status_text: HTMLSpanElement;

  _live_indicator: HTMLElement;
  _not_live_indicator: HTMLElement;
  _error_indicator: HTMLElement;

  _paused: boolean;

  _chart?: WesplotChart;

  _hostname: string;
  _socket: WebSocket;
  _data_buffer: DataRow[] = [];

  _last_data_received_time: number;

  constructor(baseHost: string) {
    this._hostname = `ws://${baseHost}/ws`;
    this._socket = new WebSocket(this._hostname);

    // Pause button
    this._pause_button = document.getElementById(
      "btn-pause"
    )! as HTMLButtonElement;
    // Pause icon in pause button
    this._pause_icon = this._pause_button.getElementsByTagName("i")[0]!;
    this._pause_button.addEventListener("click", this._handlePause.bind(this));

    this._paused = false;

    // Status text in bottom bar
    this._status_text = document.getElementById(
      "status-text"
    )! as HTMLSpanElement;

    // Live indicator
    this._live_indicator = document.getElementById("live-indicator")!;
    this._not_live_indicator = document.getElementById("not-live-indicator")!;
    this._error_indicator = document.getElementById("error-indicator")!;

    this._last_data_received_time = Date.now();

    this._socket.addEventListener("open", () => {
      console.log("Successfully Connected");
    });

    this._socket.addEventListener("close", async (event) => {
      console.log("Socket Closed Connection: ", event);
      try {
        const response = await fetch(
          `${location.protocol}//${baseHost}/errors`
        );
        const error: StreamEndedMessage = await response.json();

        if (error.StreamEnded) {
          this._set_indicator_not_live();
          this._set_status_text("Stream ended");
        }

        if (error.StreamError) {
          this.error(`Error (${error.StreamError})`);
        }
      } catch (e) {
        this.error(`Stream aborted: ${e}`);
      }
      // Disable pause after stream is ended
      this._pause_button.disabled = true;
    });

    this._socket.addEventListener("error", (error) => {
      this.error(`Websocket error: ${error}`);
    });

    this._socket.addEventListener("message", (event) => {
      this._set_indicator_live();
      // Convert to seconds and round to nearest int to prevent noise
      const time_since_last_data = Math.round(
        (Date.now() - this._last_data_received_time) / 1000
      );
      this._last_data_received_time = Date.now();
      const rows: DataRow[] = JSON.parse(event.data);
      if (this._paused) {
        this._data_buffer = this._data_buffer.concat(rows);
        this._set_indicator_not_live();
        this._set_status_text(
          `Paused: ${this._data_buffer.length} rows buffered`
        );
      } else {
        if (this._chart === undefined) {
          throw Error("Player has no registered chart");
        } else {
          this._chart.update(rows);
          this._set_status_text(
            `Live: last row received ${time_since_last_data} second(s) ago`
          );
        }
      }
    });
  }

  registerChart(chart: WesplotChart) {
    this._chart = chart;
  }

  error(status_text: string) {
    this._set_indicator_error();
    this._set_status_text(status_text);
  }

  _handlePause(_event: MouseEvent) {
    this._paused = !this._paused;
    if (this._paused) {
      this._pause_icon.classList.add("fa-play");
      this._pause_icon.classList.remove("fa-pause");
      this._set_indicator_not_live();
      this._pause_icon.title = "Resume";
    } else {
      this._pause_icon.classList.add("fa-pause");
      this._pause_icon.classList.remove("fa-play");
      this._set_indicator_live();
      this._set_status_text("Live");
      this._pause_icon.title = "Pause";
      this._chart!.update(this._data_buffer);
      this._data_buffer = [];
    }
  }

  _set_indicator_live() {
    this._live_indicator.style.display = "inline-block";
    this._not_live_indicator.style.display = "none";
    this._error_indicator.style.display = "none";
  }

  _set_indicator_not_live() {
    this._live_indicator.style.display = "none";
    this._not_live_indicator.style.display = "inline-block";
    this._error_indicator.style.display = "none";
  }

  _set_indicator_error() {
    this._live_indicator.style.display = "none";
    this._not_live_indicator.style.display = "none";
    this._error_indicator.style.display = "inline-block";
  }

  _set_status_text(status_text: string) {
    this._status_text.textContent = status_text;
  }
}
