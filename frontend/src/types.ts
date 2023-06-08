import { LimitInput } from "./limits";

export type DataRow = {
  X: number;
  Ys: number[];
};

export interface WesplotOptions {
  Title: string;
  Columns: string[];
  XLabel: string;
  YLabel: string;
  YMin?: number;
  YMax?: number;
  XMin?: number;
  XMax?: number;
  YUnit: string;
  ShowLine: boolean;
}

export interface Metadata {
  WindowSize: number;
  XIsTimestamp: boolean;
  RelativeStart: boolean;
  WesplotOptions: WesplotOptions;
}

export interface ChartButtons {
  zoom: HTMLButtonElement;
  resetzoom: HTMLButtonElement;
  pan: HTMLButtonElement;
  screenshot: HTMLButtonElement;
  settings: HTMLButtonElement;
}

export interface SettingsPanelInputs {
  cancel: HTMLButtonElement;
  save: HTMLButtonElement;
  title: HTMLInputElement;
  series_names: HTMLInputElement;
  x_min: LimitInput;
  x_max: LimitInput;
  x_label: HTMLInputElement;
  y_min: LimitInput;
  y_max: LimitInput;
  y_label: HTMLInputElement;
  y_unit: HTMLInputElement;
  relative_start: HTMLInputElement;
}

export type StreamEndedMessage = {
  StreamEnded: boolean;
  StreamError: string;
};
