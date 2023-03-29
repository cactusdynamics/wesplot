import "./styles/app.css";

import { DataRow, Metadata, StreamEndedMessage } from "./types";
import { WesplotChart } from "./wesplot-chart";

let baseHost = location.host;
if (import.meta.env.DEV) {
  // This does mean in development, we can only run one of these at a time,
  // which I think is fine.
  baseHost = `${location.hostname}:5274`;
}

async function main() {
  // Status text
  const status_text: HTMLSpanElement = document.getElementById(
    "status-text"
  )! as HTMLSpanElement;
  // Live indicator
  const live_indicator: HTMLElement =
    document.getElementById("live-indicator")!;
  const not_live_indicator: HTMLElement =
    document.getElementById("not-live-indicator")!;
  const error_indicator: HTMLElement =
    document.getElementById("error-indicator")!;

  let response: Response;
  let metadata: Metadata;

  try {
    response = await fetch(`${location.protocol}//${baseHost}/metadata`);
    metadata = await response.json();
  } catch (e) {
    live_indicator.style.display = "none";
    error_indicator.style.display = "inline-block";
    status_text.textContent = `Backend unreachable: ${e}`;
    return;
  }
  const main_panel = document.getElementById("panel")!;

  const chart = new WesplotChart(main_panel, metadata);

  // Pause button
  const pause_button: HTMLButtonElement = document.getElementById(
    "btn-pause"
  )! as HTMLButtonElement;
  const icon_elem = pause_button.getElementsByTagName("i")[0]!;

  // Pause button status
  let paused = false;
  let row_buffer: DataRow[] = [];

  const handlePause = (_event: MouseEvent) => {
    paused = !paused;
    if (paused) {
      icon_elem.classList.add("fa-play");
      icon_elem.classList.remove("fa-pause");
      live_indicator.style.display = "none";
      not_live_indicator.style.display = "inline-block";
      icon_elem.title = "Resume";
    } else {
      icon_elem.classList.add("fa-pause");
      icon_elem.classList.remove("fa-play");
      live_indicator.style.display = "inline-block";
      not_live_indicator.style.display = "none";
      icon_elem.title = "Pause";
      chart.update(row_buffer);
      row_buffer = [];
    }
  };

  pause_button.addEventListener("click", handlePause);

  const hostname = `ws://${baseHost}/ws`;
  console.log(`connecting to ${hostname}`);
  const socket = new WebSocket(hostname);

  socket.addEventListener("open", () => {
    console.log("Successfully Connected");
  });

  socket.addEventListener("close", async (event) => {
    console.log("Socket Closed Connection: ", event);
    try {
      const response = await fetch(`${location.protocol}//${baseHost}/errors`);
      const error: StreamEndedMessage = await response.json();
      live_indicator.style.display = "none";
      if (error.StreamEnded) {
        not_live_indicator.style.display = "inline-block";
      } else {
        not_live_indicator.style.display = "none";
        error_indicator.style.display = "inline-block";
        status_text.textContent = `Error (${error.StreamError})`;
      }
      pause_button.disabled = true;
    } catch (e) {
      live_indicator.style.display = "none";
      not_live_indicator.style.display = "none";
      error_indicator.style.display = "inline-block";
      status_text.textContent = `Backend unreachable: ${e}`;
      pause_button.disabled = true;
    }
  });

  socket.addEventListener("error", (error) => {
    live_indicator.style.display = "none";
    not_live_indicator.style.display = "none";
    error_indicator.style.display = "inline-block";
    status_text.textContent = `Websocket error: ${error}`;
  });

  socket.addEventListener("message", (event) => {
    const rows: DataRow[] = JSON.parse(event.data);
    if (paused) {
      row_buffer = row_buffer.concat(rows);
    } else {
      chart.update(rows);
    }
  });
}

window.addEventListener("load", main);
