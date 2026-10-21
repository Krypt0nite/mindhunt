"use client";

import { useCallback, useEffect, useRef, useState } from "react";

// Minimal types for the browser speech API (not in TypeScript's DOM lib).
interface SpeechResult {
  readonly isFinal: boolean;
  readonly 0: { readonly transcript: string };
}
interface SpeechEvent {
  readonly resultIndex: number;
  readonly results: { readonly length: number; readonly [i: number]: SpeechResult };
}
interface Recognition {
  lang: string;
  interimResults: boolean;
  continuous: boolean;
  maxAlternatives: number;
  onresult: ((e: SpeechEvent) => void) | null;
  onerror: ((e: { error: string }) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}
type RecognitionCtor = new () => Recognition;

function ctor(): RecognitionCtor | null {
  if (typeof window === "undefined") return null;
  const w = window as unknown as { SpeechRecognition?: RecognitionCtor; webkitSpeechRecognition?: RecognitionCtor };
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null;
}

const ERRORS: Record<string, string> = {
  "not-allowed": "Microphone access is blocked. Allow it in your browser settings to ask by voice.",
  "service-not-allowed": "Voice input isn't available in this browser.",
  "no-speech": "Didn't hear anything. Tap the mic and try again.",
  "audio-capture": "No microphone found.",
  network: "Voice input needs an internet connection.",
};

/**
 * Push-to-talk speech recognition. The transcript is only ever placed in
 * the question box; nothing is sent until the player confirms.
 */
export function useSpeech(onTranscript: (text: string, final: boolean) => void) {
  const [supported, setSupported] = useState(false);
  const [listening, setListening] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const rec = useRef<Recognition | null>(null);
  const cb = useRef(onTranscript);
  cb.current = onTranscript;

  useEffect(() => {
    setSupported(ctor() !== null);
    return () => rec.current?.abort();
  }, []);

  const start = useCallback(() => {
    const C = ctor();
    if (!C) return;
    rec.current?.abort();
    const r = new C();
    r.lang = navigator.language?.startsWith("en") ? navigator.language : "en-IN";
    r.interimResults = true;
    r.continuous = false;
    r.maxAlternatives = 1;
    r.onresult = (e) => {
      let text = "";
      let final = false;
      for (let i = 0; i < e.results.length; i++) {
        const res = e.results[i];
        if (!res) continue;
        text += res[0].transcript;
        final = final || res.isFinal;
      }
      cb.current(text.trim(), final);
    };
    r.onerror = (e) => {
      if (e.error !== "aborted") setError(ERRORS[e.error] ?? "Voice input stopped. Try again.");
    };
    r.onend = () => setListening(false);
    rec.current = r;
    setError(null);
    setListening(true);
    try {
      r.start();
    } catch {
      setListening(false);
      setError("Couldn't start the microphone.");
    }
  }, []);

  const stop = useCallback(() => rec.current?.stop(), []);

  return { supported, listening, error, start, stop };
}
