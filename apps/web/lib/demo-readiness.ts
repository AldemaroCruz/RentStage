export const DEMO_STEP_IDS = [
  "inventory",
  "quotes",
  "reservations",
  "billing",
  "fiscal-boundary",
] as const;

export type DemoStepID = (typeof DEMO_STEP_IDS)[number];

export type DemoReadinessInput = {
  activeResourceCount: number;
  quoteCount: number;
  acceptedQuoteCount: number;
  activeReservationCount: number;
  issuedTotal: number;
  collectedTotal: number;
  dteProviderMode?: string;
  dteEnvironment?: string;
};

export type DemoReadiness = {
  steps: Record<DemoStepID, boolean>;
  readyCount: number;
  totalCount: number;
  percent: number;
};

export function demoReadiness(input: DemoReadinessInput): DemoReadiness {
  const steps: Record<DemoStepID, boolean> = {
    inventory: input.activeResourceCount > 0,
    quotes: input.quoteCount > 0 && input.acceptedQuoteCount > 0,
    reservations: input.activeReservationCount > 0,
    billing: input.issuedTotal > 0 && input.collectedTotal > 0,
    "fiscal-boundary": input.dteProviderMode === "MOCK" && input.dteEnvironment === "TEST",
  };
  const totalCount = DEMO_STEP_IDS.length;
  const readyCount = DEMO_STEP_IDS.filter((step) => steps[step]).length;

  return {
    steps,
    readyCount,
    totalCount,
    percent: Math.round((readyCount / totalCount) * 100),
  };
}
