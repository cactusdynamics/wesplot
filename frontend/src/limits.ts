export class LimitInput {
  private _input: HTMLInputElement;
  private _is_timestamp: boolean;
  constructor(input: HTMLInputElement, is_timestamp: boolean = false) {
    this._input = input;
    this._is_timestamp = is_timestamp;

    if (is_timestamp) {
      // If we have timestamped data, limits should be set with a Unix timestamp
      this._input.type = "datetime-local";
      this._input.step = "1";
    }
  }

  // Set the value of the input
  set_value(val: number | undefined) {
    // If the value to display is undefined, the input value will say "undefined" - instead of this, show an empty string
    // Do the same for NaN
    if (val === undefined || Number.isNaN(val)) {
      this._input.value = "";

      // If the input is a timestamp, interpret the value (seconds) and format it as a date string
    } else if (this._is_timestamp) {
      this._input.value = this.formatDate(new Date(val));

      // If the value is not a timestamp and not NaN, just show the value
    } else {
      this._input.value = val.toString();
    }
  }

  get_value(): number {
    // If the limit is a timestamp, interpret the value as a date. Then, call valueOf() to get the date as a number again, but with the proper time zone
    if (this._is_timestamp) {
      return new Date(this._input.value).valueOf();
    }
    return this._input.valueAsNumber;
  }

  private padTo2Digits(num: Number) {
    return num.toString().padStart(2, "0");
  }

  private formatDate(date: Date) {
    return (
      [
        date.getFullYear(),
        this.padTo2Digits(date.getMonth() + 1),
        this.padTo2Digits(date.getDate()),
      ].join("-") +
      " " +
      [
        this.padTo2Digits(date.getHours()),
        this.padTo2Digits(date.getMinutes()),
        this.padTo2Digits(date.getSeconds()),
      ].join(":")
    );
  }
}
