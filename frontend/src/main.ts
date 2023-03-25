import "./app.css";
import "@fortawesome/fontawesome-free/css/fontawesome.css";
import "@fortawesome/fontawesome-free/css/solid.css";
import { DataRow, Metadata } from "./types";
import { WesplotChart } from "./wesplot-chart";

async function main() {
  const response = await fetch(`http://${location.hostname}:8080/metadata`);
  const metadata: Metadata = await response.json();
  const ctx = document.getElementById("myChart")! as HTMLCanvasElement;

  const chart = new WesplotChart(ctx, metadata);

  const hostname = `ws://${location.hostname}:8080/ws`;
  console.log(`connecting to ${hostname}`);
  const socket = new WebSocket(hostname);

  socket.addEventListener("open", () => {
    console.log("Successfully Connected");
  });

  socket.addEventListener("close", (event) => {
    console.log("Socket Closed Connection: ", event);
  });

  socket.addEventListener("error", (error) => {
    console.log("Socket Error: ", error);
  });

  socket.addEventListener("message", (event) => {
    const rows: DataRow[] = JSON.parse(event.data);
    chart.update(rows);
  });
}

window.addEventListener("load", main);
