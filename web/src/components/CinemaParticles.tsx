import React, { useMemo } from 'react';
import styles from './CinemaParticles.module.css';

interface Props {
  color: string;
}

interface Particle {
  id: number;
  x: number;
  y: number;
  size: number;
  duration: number;
  delay: number;
  opacity: number;
}

// Deterministic pseudo-random based on seed
function seededRandom(seed: number): number {
  const x = Math.sin(seed * 127.1 + 311.7) * 43758.5453;
  return x - Math.floor(x);
}

export const CinemaParticles: React.FC<Props> = ({ color }) => {
  // Generate particles once, deterministic
  const particles = useMemo<Particle[]>(() => {
    return Array.from({ length: 18 }, (_, i) => ({
      id: i,
      x: seededRandom(i * 3) * 100,
      y: seededRandom(i * 3 + 1) * 100,
      size: 2 + seededRandom(i * 3 + 2) * 4,
      duration: 15 + seededRandom(i * 7) * 25,
      delay: seededRandom(i * 11) * -30,
      opacity: 0.15 + seededRandom(i * 13) * 0.25,
    }));
  }, []);

  return (
    <div className={styles.container}>
      {particles.map(p => (
        <div
          key={p.id}
          className={styles.particle}
          style={{
            left: `${p.x}%`,
            top: `${p.y}%`,
            width: `${p.size}px`,
            height: `${p.size}px`,
            backgroundColor: color,
            opacity: p.opacity,
            animationDuration: `${p.duration}s`,
            animationDelay: `${p.delay}s`,
          }}
        />
      ))}
    </div>
  );
};
