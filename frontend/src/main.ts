import { Player } from "./player";
import "./styles/app.css";

import { Metadata } from "./types";
import { WesplotChart } from "./wesplot-chart";

let baseHost = location.host;
if (import.meta.env.DEV) {
  // This does mean in development, we can only run one of these at a time,
  // which I think is fine.
  baseHost = `${location.hostname}:5274`;
}

async function main() {
  const player = new Player();
  let response: Response;
  let metadata: Metadata;

  try {
    response = await fetch(`${location.protocol}//${baseHost}/metadata`);
    metadata = await response.json();
  } catch (e) {
    player.handleError(`Backend unreachable: ${e}`);
    return;
  }
  const main_panel = document.getElementById("panel")!;

  const chart = new WesplotChart(main_panel, metadata);

  player.registerChart(chart);
  player.connectToWebsocket(baseHost);
}

window.addEventListener("load", main);
