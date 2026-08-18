export const LOCAL_DEMO_CREDENTIALS = Object.freeze({
  email: "owner@rentstage.local",
  password: "RentStage123!",
});

const EMPTY_LOGIN_DEFAULTS = Object.freeze({ email: "", password: "" });

export function authEmulatorEnabled(configuredValue: string | undefined): boolean {
  return (configuredValue ?? "true").trim().toLowerCase() === "true";
}

export const AUTH_EMULATOR_ENABLED = authEmulatorEnabled(
  process.env.NEXT_PUBLIC_USE_AUTH_EMULATOR,
);

export function loginDefaults(configuredValue: string | undefined) {
  return authEmulatorEnabled(configuredValue)
    ? LOCAL_DEMO_CREDENTIALS
    : EMPTY_LOGIN_DEFAULTS;
}

export const LOGIN_DEFAULTS = loginDefaults(
  process.env.NEXT_PUBLIC_USE_AUTH_EMULATOR,
);
