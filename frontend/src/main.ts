import "./styles/app.css";

import { DataRow, Metadata } from "./types";
import { WesplotChart } from "./wesplot-chart";

async function main() {
  const response = await fetch(`http://${location.hostname}:8080/metadata`);
  const metadata: Metadata = await response.json();
  const main_panel = document.getElementById("panel")!;

  const chart = new WesplotChart(main_panel, metadata);

  // Pause button
  const pause_button: HTMLButtonElement = document.getElementById(
    "btn-pause"
  ) as HTMLButtonElement;
  const icon_elem = pause_button.getElementsByTagName("i")[0]!;

  // Pause button status
  let paused = false;
  let row_buffer: DataRow[] = [];

  const handlePause = (_event: MouseEvent) => {
    paused = !paused;
    if (paused) {
      icon_elem.classList.add("fa-play");
      icon_elem.classList.remove("fa-pause");
      icon_elem.title = "Resume";
    } else {
      icon_elem.classList.add("fa-pause");
      icon_elem.classList.remove("fa-play");
      icon_elem.title = "Pause";
      chart.update(row_buffer);
      row_buffer = [];
    }
  };

  pause_button.addEventListener("click", handlePause);

  const hostname = `ws://${location.hostname}:8080/ws`;
  console.log(`connecting to ${hostname}`);
  const socket = new WebSocket(hostname);

  socket.addEventListener("open", () => {
    console.log("Successfully Connected");
  });

  socket.addEventListener("close", async (event) => {
    console.log("Socket Closed Connection: ", event);
    try {
      const response = await fetch(`http://${location.hostname}:8080/errors`);
      const error: unknown = await response.json();
      console.log(error);
    } catch (e) {
      console.log("Backend died");
    }
  });

  socket.addEventListener("error", (error) => {
    console.log("Socket Error: ", error);
  });

  socket.addEventListener("message", (event) => {
    const rows: DataRow[] = JSON.parse(event.data);
    if (paused) {
      row_buffer.concat(rows);
    } else {
      chart.update(rows);
    }
  });
}

window.addEventListener("load", main);
