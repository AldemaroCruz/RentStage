"use client";

import { getApp, getApps, initializeApp } from "firebase/app";
import {
  Auth,
  connectAuthEmulator,
  getAuth,
  inMemoryPersistence,
  setPersistence,
} from "firebase/auth";

let configured = false;
let persistencePromise: Promise<void> | null = null;

export function getFirebaseAuth(): Auth {
  const app = getApps().length > 0
    ? getApp()
    : initializeApp({
        apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY || "rentstage-local-api-key",
        authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN || "demo-rentstage.firebaseapp.com",
        projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID || "demo-rentstage",
      });
  const auth = getAuth(app);

  if (typeof window !== "undefined" && !configured) {
    const useEmulator = (process.env.NEXT_PUBLIC_USE_AUTH_EMULATOR || "true") === "true";
    if (useEmulator) {
      const port = process.env.NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_PORT || "9099";
      connectAuthEmulator(auth, `http://${window.location.hostname}:${port}`, { disableWarnings: true });
    }
    persistencePromise = setPersistence(auth, inMemoryPersistence);
    configured = true;
  }
  return auth;
}

export async function ensureFirebasePersistence(): Promise<void> {
  getFirebaseAuth();
  if (persistencePromise) await persistencePromise;
}
