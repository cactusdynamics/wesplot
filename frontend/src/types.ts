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
  x_min: HTMLInputElement;
  x_max: HTMLInputElement;
  x_label: HTMLInputElement;
  y_min: HTMLInputElement;
  y_max: HTMLInputElement;
  y_label: HTMLInputElement;
  y_unit: HTMLInputElement;
  relative_start: HTMLInputElement;
}

export type StreamEndedMessage = {
  StreamEnded: boolean;
  StreamError: string;
};
