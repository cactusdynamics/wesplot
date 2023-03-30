import { Player } from "./player";
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
  const player = new Player(baseHost);
  let response: Response;
  let metadata: Metadata;

  try {
    response = await fetch(`${location.protocol}//${baseHost}/metadata`);
    metadata = await response.json();
  } catch (e) {
    player.error(`Backend unreachable: ${e}`);
    return;
  }
  const main_panel = document.getElementById("panel")!;

  const chart = new WesplotChart(main_panel, metadata);

  player.registerChart(chart);
}

window.addEventListener("load", main);
