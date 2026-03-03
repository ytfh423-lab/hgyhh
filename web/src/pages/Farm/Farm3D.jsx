import React, { useRef, useMemo, useState, useEffect } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { OrbitControls, Text, Html, RoundedBox } from '@react-three/drei';
import * as THREE from 'three';

// ==================== Constants ====================
const PLOT_SIZE = 1.6;
const PLOT_GAP = 0.3;
const PLOT_HEIGHT = 0.25;
const COLS = 5;

// ==================== Color Palettes ====================
const COLORS = {
  grass: '#4ade80',
  grassDark: '#22c55e',
  soil: '#92400e',
  soilLight: '#a16207',
  soilDry: '#78350f',
  water: '#38bdf8',
  fence: '#d97706',
  fencePost: '#92400e',
  trunk: '#78350f',
  leaves: '#16a34a',
  leavesDark: '#15803d',
  flower: '#f472b6',
  fruit: '#ef4444',
  fruitOrange: '#f97316',
  mature: '#eab308',
  wilt: '#a1a1aa',
  danger: '#ef4444',
  sky: '#e0f2fe',
  path: '#d6d3d1',
};

const CROP_COLORS = {
  watermelon: { body: '#22c55e', stripe: '#15803d', fruit: '#ef4444' },
  strawberry: { body: '#16a34a', fruit: '#ef4444', seed: '#fbbf24' },
  carrot: { body: '#22c55e', fruit: '#f97316' },
  corn: { body: '#16a34a', fruit: '#eab308' },
  rice: { body: '#a3e635', fruit: '#fde047' },
  potato: { body: '#22c55e', fruit: '#a16207' },
  tomato: { body: '#16a34a', fruit: '#ef4444' },
  pumpkin: { body: '#16a34a', fruit: '#f97316' },
  default: { body: '#22c55e', fruit: '#84cc16' },
};

// ==================== Helpers ====================
const getGridPos = (index, totalPlots) => {
  const cols = Math.min(COLS, totalPlots);
  const row = Math.floor(index / cols);
  const col = index % cols;
  const totalCols = Math.min(cols, totalPlots);
  const totalRows = Math.ceil(totalPlots / cols);
  const x = (col - (totalCols - 1) / 2) * (PLOT_SIZE + PLOT_GAP);
  const z = (row - (totalRows - 1) / 2) * (PLOT_SIZE + PLOT_GAP);
  return [x, 0, z];
};

const getCropColor = (cropType) => {
  const key = (cropType || '').toLowerCase();
  for (const k of Object.keys(CROP_COLORS)) {
    if (key.includes(k)) return CROP_COLORS[k];
  }
  return CROP_COLORS.default;
};

// ==================== Ground ====================
const Ground = ({ totalPlots }) => {
  const cols = Math.min(COLS, totalPlots);
  const rows = Math.ceil(totalPlots / cols);
  const w = cols * (PLOT_SIZE + PLOT_GAP) + 3;
  const h = rows * (PLOT_SIZE + PLOT_GAP) + 3;

  return (
    <group>
      {/* Main grass ground */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.05, 0]} receiveShadow>
        <planeGeometry args={[w + 4, h + 4]} />
        <meshStandardMaterial color={COLORS.grass} />
      </mesh>
      {/* Darker edge */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.06, 0]}>
        <planeGeometry args={[w + 6, h + 6]} />
        <meshStandardMaterial color={COLORS.grassDark} />
      </mesh>
      {/* Path */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.04, h / 2 + 1.2]}>
        <planeGeometry args={[2.5, 1.5]} />
        <meshStandardMaterial color={COLORS.path} />
      </mesh>
    </group>
  );
};

// ==================== Fence ====================
const Fence = ({ totalPlots }) => {
  const cols = Math.min(COLS, totalPlots);
  const rows = Math.ceil(totalPlots / cols);
  const w = cols * (PLOT_SIZE + PLOT_GAP) + 2;
  const h = rows * (PLOT_SIZE + PLOT_GAP) + 2;
  const posts = [];
  const rails = [];

  // Create fence posts and rails around the perimeter
  const spacing = 1.8;
  const sides = [
    { start: [-w / 2, 0, -h / 2], dir: [1, 0, 0], len: w },
    { start: [w / 2, 0, -h / 2], dir: [0, 0, 1], len: h },
    { start: [-w / 2, 0, h / 2], dir: [1, 0, 0], len: w },
    { start: [-w / 2, 0, -h / 2], dir: [0, 0, 1], len: h },
  ];

  sides.forEach((side, si) => {
    const count = Math.floor(side.len / spacing) + 1;
    for (let i = 0; i < count; i++) {
      const t = i / (count - 1 || 1);
      const x = side.start[0] + side.dir[0] * side.len * t;
      const z = side.start[2] + side.dir[2] * side.len * t;
      // Skip gate area
      if (si === 2 && Math.abs(x) < 1.5) continue;
      posts.push([x, 0.3, z]);
    }
  });

  return (
    <group>
      {posts.map((pos, i) => (
        <group key={`post-${i}`} position={pos}>
          {/* Post */}
          <mesh castShadow>
            <cylinderGeometry args={[0.06, 0.08, 0.7, 6]} />
            <meshStandardMaterial color={COLORS.fencePost} />
          </mesh>
          {/* Top */}
          <mesh position={[0, 0.38, 0]}>
            <sphereGeometry args={[0.08, 6, 6]} />
            <meshStandardMaterial color={COLORS.fence} />
          </mesh>
        </group>
      ))}
      {/* Horizontal rails */}
      {posts.map((pos, i) => {
        if (i === posts.length - 1) return null;
        const next = posts[i + 1];
        if (!next) return null;
        const dx = next[0] - pos[0];
        const dz = next[2] - pos[2];
        const dist = Math.sqrt(dx * dx + dz * dz);
        if (dist > spacing * 1.5) return null;
        const mx = (pos[0] + next[0]) / 2;
        const mz = (pos[2] + next[2]) / 2;
        const angle = Math.atan2(dx, dz);
        return (
          <group key={`rail-${i}`}>
            <mesh position={[mx, 0.45, mz]} rotation={[0, angle, Math.PI / 2]}>
              <cylinderGeometry args={[0.03, 0.03, dist, 4]} />
              <meshStandardMaterial color={COLORS.fence} />
            </mesh>
            <mesh position={[mx, 0.2, mz]} rotation={[0, angle, Math.PI / 2]}>
              <cylinderGeometry args={[0.03, 0.03, dist, 4]} />
              <meshStandardMaterial color={COLORS.fence} />
            </mesh>
          </group>
        );
      })}
    </group>
  );
};

// ==================== Soil Plot ====================
const SoilPlot = ({ position, status, onClick }) => {
  const meshRef = useRef();
  const [hovered, setHovered] = useState(false);

  const soilColor = useMemo(() => {
    if (status === 4) return COLORS.soilDry;
    if (status === 3) return COLORS.danger;
    return hovered ? COLORS.soilLight : COLORS.soil;
  }, [status, hovered]);

  return (
    <group position={position}>
      {/* Soil block */}
      <RoundedBox
        ref={meshRef}
        args={[PLOT_SIZE, PLOT_HEIGHT, PLOT_SIZE]}
        radius={0.05}
        position={[0, PLOT_HEIGHT / 2, 0]}
        castShadow
        receiveShadow
        onClick={onClick}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
      >
        <meshStandardMaterial color={soilColor} />
      </RoundedBox>
      {/* Soil lines (furrows) */}
      {[-0.45, -0.15, 0.15, 0.45].map((offset, i) => (
        <mesh key={i} position={[0, PLOT_HEIGHT + 0.01, offset]} rotation={[-Math.PI / 2, 0, 0]}>
          <planeGeometry args={[PLOT_SIZE - 0.2, 0.06]} />
          <meshStandardMaterial color='#7c2d12' transparent opacity={0.3} />
        </mesh>
      ))}
      {/* Water shimmer for watered plots */}
      {status === 1 && (
        <mesh position={[0, PLOT_HEIGHT + 0.02, 0]} rotation={[-Math.PI / 2, 0, 0]}>
          <planeGeometry args={[PLOT_SIZE - 0.1, PLOT_SIZE - 0.1]} />
          <meshStandardMaterial color={COLORS.water} transparent opacity={0.1} />
        </mesh>
      )}
    </group>
  );
};

// ==================== Crop Models ====================
const EmptyPlotSign = ({ position }) => {
  return (
    <group position={[position[0], PLOT_HEIGHT + 0.01, position[2]]}>
      {/* Small sign */}
      <mesh position={[0, 0.25, 0]}>
        <boxGeometry args={[0.03, 0.5, 0.03]} />
        <meshStandardMaterial color={COLORS.fencePost} />
      </mesh>
      <mesh position={[0, 0.5, 0]}>
        <boxGeometry args={[0.3, 0.2, 0.02]} />
        <meshStandardMaterial color='#fef3c7' />
      </mesh>
    </group>
  );
};

const GrowingCrop = ({ position, progress, cropType, fertilized }) => {
  const groupRef = useRef();
  const colors = getCropColor(cropType);
  const scale = 0.3 + (progress / 100) * 0.7;
  const stemHeight = 0.2 + (progress / 100) * 0.5;

  useFrame((state) => {
    if (groupRef.current) {
      groupRef.current.rotation.y = Math.sin(state.clock.elapsedTime * 0.5 + position[0]) * 0.05;
    }
  });

  return (
    <group ref={groupRef} position={[position[0], PLOT_HEIGHT, position[2]]}>
      {/* Multiple stems in a cluster */}
      {[[-0.2, -0.2], [0.2, -0.2], [0, 0.15], [-0.3, 0.1], [0.3, 0.1]].map(([ox, oz], i) => {
        const s = scale * (0.8 + Math.random() * 0.2);
        const h = stemHeight * (0.7 + i * 0.08);
        return (
          <group key={i} position={[ox * scale, 0, oz * scale]}>
            {/* Stem */}
            <mesh position={[0, h / 2, 0]} castShadow>
              <cylinderGeometry args={[0.02 * s, 0.03 * s, h, 5]} />
              <meshStandardMaterial color='#4d7c0f' />
            </mesh>
            {/* Leaves */}
            <mesh position={[0, h * 0.7, 0]} castShadow>
              <sphereGeometry args={[0.12 * s, 6, 6]} />
              <meshStandardMaterial color={colors.body} />
            </mesh>
            {/* Top leaf/bud */}
            <mesh position={[0, h, 0]} castShadow>
              <coneGeometry args={[0.08 * s, 0.15 * s, 5]} />
              <meshStandardMaterial color={colors.body} />
            </mesh>
          </group>
        );
      })}
      {/* Fertilizer sparkle */}
      {fertilized === 1 && (
        <FertilizerEffect position={[0, stemHeight + 0.2, 0]} />
      )}
    </group>
  );
};

const FertilizerEffect = ({ position }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.rotation.y = state.clock.elapsedTime * 2;
      ref.current.position.y = position[1] + Math.sin(state.clock.elapsedTime * 3) * 0.05;
    }
  });
  return (
    <group ref={ref} position={position}>
      {[0, 1, 2, 3].map(i => (
        <mesh key={i} position={[Math.cos(i * Math.PI / 2) * 0.15, 0, Math.sin(i * Math.PI / 2) * 0.15]}>
          <octahedronGeometry args={[0.04, 0]} />
          <meshStandardMaterial color='#67e8f9' emissive='#22d3ee' emissiveIntensity={0.5} />
        </mesh>
      ))}
    </group>
  );
};

const MatureCrop = ({ position, cropType }) => {
  const groupRef = useRef();
  const colors = getCropColor(cropType);

  useFrame((state) => {
    if (groupRef.current) {
      groupRef.current.rotation.y = Math.sin(state.clock.elapsedTime * 0.3) * 0.03;
    }
  });

  return (
    <group ref={groupRef} position={[position[0], PLOT_HEIGHT, position[2]]}>
      {/* Full grown crop cluster */}
      {[[-0.25, -0.2], [0.2, -0.25], [0, 0.2], [-0.3, 0.15], [0.25, 0.15]].map(([ox, oz], i) => (
        <group key={i} position={[ox, 0, oz]}>
          {/* Stem */}
          <mesh position={[0, 0.3, 0]} castShadow>
            <cylinderGeometry args={[0.025, 0.04, 0.6, 5]} />
            <meshStandardMaterial color='#4d7c0f' />
          </mesh>
          {/* Leaves */}
          <mesh position={[0.08, 0.35, 0]} rotation={[0, 0, 0.5]} castShadow>
            <boxGeometry args={[0.18, 0.04, 0.08]} />
            <meshStandardMaterial color={colors.body} />
          </mesh>
          <mesh position={[-0.08, 0.28, 0]} rotation={[0, 0, -0.5]} castShadow>
            <boxGeometry args={[0.18, 0.04, 0.08]} />
            <meshStandardMaterial color={colors.body} />
          </mesh>
          {/* Fruit */}
          <mesh position={[0, 0.55, 0]} castShadow>
            <sphereGeometry args={[0.12, 8, 8]} />
            <meshStandardMaterial color={colors.fruit} />
          </mesh>
        </group>
      ))}
      {/* Sparkle effect - ready to harvest */}
      <HarvestSparkle />
    </group>
  );
};

const HarvestSparkle = () => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.rotation.y = state.clock.elapsedTime;
      const s = 1 + Math.sin(state.clock.elapsedTime * 2) * 0.2;
      ref.current.scale.set(s, s, s);
    }
  });
  return (
    <group ref={ref} position={[0, 0.8, 0]}>
      {[0, 1, 2, 3, 4, 5].map(i => (
        <mesh key={i} position={[Math.cos(i * Math.PI / 3) * 0.25, Math.sin(i * 2) * 0.1, Math.sin(i * Math.PI / 3) * 0.25]}>
          <octahedronGeometry args={[0.03, 0]} />
          <meshStandardMaterial color={COLORS.mature} emissive={COLORS.mature} emissiveIntensity={1} />
        </mesh>
      ))}
    </group>
  );
};

const EventCrop = ({ position, eventType, cropType }) => {
  const ref = useRef();

  useFrame((state) => {
    if (ref.current) {
      ref.current.position.x = position[0] + Math.sin(state.clock.elapsedTime * 8) * 0.02;
    }
  });

  const isDrought = eventType === 'drought';

  return (
    <group ref={ref} position={[position[0], PLOT_HEIGHT, position[2]]}>
      {/* Wilted stems */}
      {[[-0.2, -0.15], [0.15, -0.2], [0, 0.15]].map(([ox, oz], i) => (
        <group key={i} position={[ox, 0, oz]} rotation={[0.3, 0, i * 0.5]}>
          <mesh position={[0, 0.2, 0]} castShadow>
            <cylinderGeometry args={[0.02, 0.03, 0.4, 5]} />
            <meshStandardMaterial color={isDrought ? '#a16207' : '#4d7c0f'} />
          </mesh>
          <mesh position={[0, 0.35, 0]} castShadow>
            <sphereGeometry args={[0.08, 5, 5]} />
            <meshStandardMaterial color={isDrought ? '#d97706' : '#84cc16'} />
          </mesh>
        </group>
      ))}
      {/* Event indicator */}
      {isDrought ? (
        <DroughtEffect />
      ) : (
        <BugEffect />
      )}
    </group>
  );
};

const DroughtEffect = () => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.rotation.y = state.clock.elapsedTime * 0.5;
    }
  });
  return (
    <group ref={ref} position={[0, 0.6, 0]}>
      {/* Heat waves */}
      {[0, 1, 2].map(i => (
        <mesh key={i} position={[Math.cos(i * Math.PI * 2 / 3) * 0.2, i * 0.1, Math.sin(i * Math.PI * 2 / 3) * 0.2]}>
          <torusGeometry args={[0.08, 0.02, 4, 8]} />
          <meshStandardMaterial color='#fbbf24' transparent opacity={0.6} emissive='#f59e0b' emissiveIntensity={0.5} />
        </mesh>
      ))}
    </group>
  );
};

const BugEffect = () => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.children.forEach((child, i) => {
        child.position.x = Math.cos(state.clock.elapsedTime * 3 + i * 2) * 0.3;
        child.position.z = Math.sin(state.clock.elapsedTime * 3 + i * 2) * 0.3;
        child.position.y = 0.5 + Math.sin(state.clock.elapsedTime * 5 + i) * 0.1;
      });
    }
  });
  return (
    <group ref={ref}>
      {[0, 1, 2, 3].map(i => (
        <mesh key={i}>
          <sphereGeometry args={[0.03, 4, 4]} />
          <meshStandardMaterial color='#1a1a1a' />
        </mesh>
      ))}
    </group>
  );
};

const WiltCrop = ({ position, cropType }) => {
  return (
    <group position={[position[0], PLOT_HEIGHT, position[2]]}>
      {/* Drooping dead-looking stems */}
      {[[-0.2, -0.15], [0.15, -0.2], [0, 0.15]].map(([ox, oz], i) => (
        <group key={i} position={[ox, 0, oz]} rotation={[0.6, 0, i * 0.8]}>
          <mesh position={[0, 0.12, 0]} castShadow>
            <cylinderGeometry args={[0.015, 0.025, 0.25, 4]} />
            <meshStandardMaterial color='#78716c' />
          </mesh>
          <mesh position={[0, 0.22, 0]} castShadow>
            <sphereGeometry args={[0.06, 4, 4]} />
            <meshStandardMaterial color={COLORS.wilt} />
          </mesh>
        </group>
      ))}
      {/* Danger indicator */}
      <WarningSign position={[0, 0.7, 0]} />
    </group>
  );
};

const WarningSign = ({ position }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.position.y = position[1] + Math.sin(state.clock.elapsedTime * 3) * 0.05;
    }
  });
  return (
    <group ref={ref} position={position}>
      <mesh>
        <coneGeometry args={[0.08, 0.12, 3]} />
        <meshStandardMaterial color={COLORS.danger} emissive={COLORS.danger} emissiveIntensity={0.3} />
      </mesh>
    </group>
  );
};

// ==================== Dog 3D Model ====================
const FarmDog = ({ dogData, totalPlots }) => {
  const groupRef = useRef();
  const tailRef = useRef();
  const cols = Math.min(COLS, totalPlots);
  const rows = Math.ceil(totalPlots / cols);
  const patrolRadius = Math.max(cols, rows) * (PLOT_SIZE + PLOT_GAP) / 2 + 0.5;

  useFrame((state) => {
    if (groupRef.current) {
      const t = state.clock.elapsedTime * 0.3;
      groupRef.current.position.x = Math.cos(t) * patrolRadius;
      groupRef.current.position.z = Math.sin(t) * patrolRadius;
      groupRef.current.rotation.y = -t + Math.PI / 2;
    }
    if (tailRef.current) {
      tailRef.current.rotation.z = Math.sin(state.clock.elapsedTime * 8) * 0.4;
    }
  });

  const isAdult = dogData?.level === 2;
  const bodyScale = isAdult ? 1.2 : 0.8;

  return (
    <group ref={groupRef} position={[patrolRadius, 0, 0]}>
      <group scale={bodyScale}>
        {/* Body */}
        <mesh position={[0, 0.25, 0]} castShadow>
          <boxGeometry args={[0.25, 0.2, 0.4]} />
          <meshStandardMaterial color={isAdult ? '#92400e' : '#d97706'} />
        </mesh>
        {/* Head */}
        <mesh position={[0, 0.35, 0.22]} castShadow>
          <boxGeometry args={[0.2, 0.18, 0.2]} />
          <meshStandardMaterial color={isAdult ? '#a16207' : '#eab308'} />
        </mesh>
        {/* Snout */}
        <mesh position={[0, 0.32, 0.35]} castShadow>
          <boxGeometry args={[0.1, 0.08, 0.1]} />
          <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
        </mesh>
        {/* Nose */}
        <mesh position={[0, 0.33, 0.41]}>
          <sphereGeometry args={[0.03, 6, 6]} />
          <meshStandardMaterial color='#1c1917' />
        </mesh>
        {/* Eyes */}
        <mesh position={[-0.06, 0.39, 0.32]}>
          <sphereGeometry args={[0.025, 6, 6]} />
          <meshStandardMaterial color='#1c1917' />
        </mesh>
        <mesh position={[0.06, 0.39, 0.32]}>
          <sphereGeometry args={[0.025, 6, 6]} />
          <meshStandardMaterial color='#1c1917' />
        </mesh>
        {/* Ears */}
        <mesh position={[-0.1, 0.44, 0.2]} rotation={[0, 0, -0.3]}>
          <boxGeometry args={[0.08, 0.12, 0.06]} />
          <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
        </mesh>
        <mesh position={[0.1, 0.44, 0.2]} rotation={[0, 0, 0.3]}>
          <boxGeometry args={[0.08, 0.12, 0.06]} />
          <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
        </mesh>
        {/* Legs */}
        {[[-0.08, -0.15], [0.08, -0.15], [-0.08, 0.15], [0.08, 0.15]].map(([x, z], i) => (
          <mesh key={i} position={[x, 0.08, z]} castShadow>
            <boxGeometry args={[0.06, 0.16, 0.06]} />
            <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
          </mesh>
        ))}
        {/* Tail */}
        <group ref={tailRef} position={[0, 0.35, -0.2]}>
          <mesh position={[0, 0.08, -0.05]} rotation={[0.5, 0, 0]}>
            <cylinderGeometry args={[0.025, 0.015, 0.18, 5]} />
            <meshStandardMaterial color={isAdult ? '#a16207' : '#eab308'} />
          </mesh>
        </group>
      </group>
      {/* Name label */}
      <Html position={[0, 0.65 * bodyScale, 0]} center distanceFactor={8} style={{ pointerEvents: 'none' }}>
        <div style={{
          background: 'rgba(0,0,0,0.6)', color: 'white', padding: '2px 8px',
          borderRadius: 4, fontSize: 11, whiteSpace: 'nowrap', fontWeight: 600,
        }}>
          {isAdult ? '🐕' : '🐶'} {dogData?.name}
        </div>
      </Html>
    </group>
  );
};

// ==================== Decorative Trees ====================
const DecoTree = ({ position, scale = 1 }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.rotation.y = Math.sin(state.clock.elapsedTime * 0.5 + position[0]) * 0.05;
    }
  });

  return (
    <group ref={ref} position={position} scale={scale}>
      {/* Trunk */}
      <mesh position={[0, 0.35, 0]} castShadow>
        <cylinderGeometry args={[0.06, 0.1, 0.7, 6]} />
        <meshStandardMaterial color={COLORS.trunk} />
      </mesh>
      {/* Foliage layers */}
      <mesh position={[0, 0.75, 0]} castShadow>
        <coneGeometry args={[0.4, 0.5, 8]} />
        <meshStandardMaterial color={COLORS.leaves} />
      </mesh>
      <mesh position={[0, 1.0, 0]} castShadow>
        <coneGeometry args={[0.3, 0.4, 8]} />
        <meshStandardMaterial color={COLORS.leavesDark} />
      </mesh>
      <mesh position={[0, 1.2, 0]} castShadow>
        <coneGeometry args={[0.2, 0.35, 8]} />
        <meshStandardMaterial color={COLORS.leaves} />
      </mesh>
    </group>
  );
};

// ==================== Water Drops Animation ====================
const WaterDrops = ({ position }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.children.forEach((drop, i) => {
        const t = (state.clock.elapsedTime * 2 + i * 0.5) % 1;
        drop.position.y = 0.8 - t * 0.8;
        drop.scale.setScalar(1 - t);
        drop.material.opacity = 1 - t;
      });
    }
  });
  return (
    <group ref={ref} position={position}>
      {[0, 1, 2].map(i => (
        <mesh key={i} position={[Math.cos(i * 2) * 0.1, 0, Math.sin(i * 2) * 0.1]}>
          <sphereGeometry args={[0.04, 6, 6]} />
          <meshStandardMaterial color={COLORS.water} transparent opacity={0.8} />
        </mesh>
      ))}
    </group>
  );
};

// ==================== Plot Label (HTML overlay) ====================
const PlotLabel = ({ position, plot, t, onAction }) => {
  const statusText = {
    0: '空地',
    1: '生长中',
    2: '已成熟 ✨',
    3: plot?.event_type === 'drought' ? '干旱!' : '虫害!',
    4: '枯萎!',
  };

  return (
    <Html position={[position[0], 1.2, position[2]]} center distanceFactor={10}
      style={{ pointerEvents: 'auto' }}>
      <div style={{
        background: 'rgba(255,255,255,0.92)', borderRadius: 8, padding: '6px 10px',
        boxShadow: '0 2px 12px rgba(0,0,0,0.15)', minWidth: 100, textAlign: 'center',
        border: `2px solid ${plot.status === 2 ? '#22c55e' : plot.status === 3 || plot.status === 4 ? '#ef4444' : '#e5e7eb'}`,
        backdropFilter: 'blur(4px)',
      }}>
        <div style={{ fontWeight: 700, fontSize: 12, marginBottom: 2 }}>
          {plot.crop_emoji || '📍'} {plot.plot_index + 1}{t('号地')}
        </div>
        <div style={{
          fontSize: 11,
          color: plot.status === 2 ? '#16a34a' : plot.status >= 3 ? '#dc2626' : '#6b7280',
          fontWeight: 600,
        }}>
          {plot.crop_name ? `${plot.crop_name} · ` : ''}{statusText[plot.status] || ''}
        </div>
        {plot.status === 1 && (
          <div style={{ marginTop: 3 }}>
            <div style={{
              background: '#e5e7eb', borderRadius: 4, height: 6, overflow: 'hidden',
            }}>
              <div style={{
                background: 'linear-gradient(90deg, #22c55e, #84cc16)',
                height: '100%', width: `${plot.progress}%`, borderRadius: 4,
                transition: 'width 0.3s',
              }} />
            </div>
            <div style={{ fontSize: 10, color: '#9ca3af', marginTop: 1 }}>{plot.progress}%</div>
          </div>
        )}
        {/* Action buttons */}
        {plot.status === 1 && (
          <div style={{ display: 'flex', gap: 4, marginTop: 4, justifyContent: 'center' }}>
            <button onClick={() => onAction('water', plot.plot_index)} style={btnStyle('#38bdf8')}>
              💧
            </button>
            {plot.fertilized === 0 && (
              <button onClick={() => onAction('fertilize', plot.plot_index)} style={btnStyle('#67e8f9')}>
                🧴
              </button>
            )}
          </div>
        )}
        {plot.status === 3 && plot.event_type === 'drought' && (
          <button onClick={() => onAction('water', plot.plot_index)}
            style={{ ...btnStyle('#ef4444'), marginTop: 4, width: '100%' }}>
            💧 {t('浇水')}
          </button>
        )}
        {plot.status === 3 && plot.event_type !== 'drought' && (
          <button onClick={() => onAction('treat', plot.plot_index)}
            style={{ ...btnStyle('#f59e0b'), marginTop: 4, width: '100%' }}>
            💊 {t('治疗')}
          </button>
        )}
        {plot.status === 4 && (
          <button onClick={() => onAction('water', plot.plot_index)}
            style={{ ...btnStyle('#ef4444'), marginTop: 4, width: '100%' }}>
            💧 {t('浇水')}
          </button>
        )}
      </div>
    </Html>
  );
};

const btnStyle = (bg) => ({
  background: bg, border: 'none', borderRadius: 4, padding: '3px 8px',
  fontSize: 12, cursor: 'pointer', color: 'white', fontWeight: 600,
  transition: 'transform 0.1s',
});

// ==================== Scene Setup ====================
const SceneSetup = ({ totalPlots }) => {
  const { camera } = useThree();
  const cols = Math.min(COLS, totalPlots);
  const rows = Math.ceil(totalPlots / cols);
  const maxDim = Math.max(cols, rows);

  useEffect(() => {
    const dist = maxDim * 1.8 + 3;
    camera.position.set(dist * 0.7, dist * 0.6, dist * 0.7);
    camera.lookAt(0, 0, 0);
  }, [camera, maxDim]);

  return null;
};

// ==================== Main 3D Farm Component ====================
const Farm3DView = ({ farmData, doAction, t, selectedPlotIndex, setSelectedPlotIndex }) => {
  const plots = farmData?.plots || [];
  const totalPlots = plots.length;

  const handlePlotAction = (action, plotIndex) => {
    if (action === 'water') {
      doAction('/api/farm/water', { plot_index: plotIndex });
    } else if (action === 'fertilize') {
      doAction('/api/farm/fertilize', { plot_index: plotIndex });
    } else if (action === 'treat') {
      doAction('/api/farm/treat', { plot_index: plotIndex });
    }
  };

  const handlePlotClick = (plotIndex) => {
    setSelectedPlotIndex(selectedPlotIndex === plotIndex ? null : plotIndex);
  };

  // Tree positions - decorative
  const cols = Math.min(COLS, totalPlots);
  const rows = Math.ceil(totalPlots / cols);
  const farmW = cols * (PLOT_SIZE + PLOT_GAP) / 2 + 2;
  const farmH = rows * (PLOT_SIZE + PLOT_GAP) / 2 + 2;

  const treePositions = useMemo(() => [
    [-farmW - 0.5, 0, -farmH - 0.5],
    [farmW + 0.5, 0, -farmH - 0.5],
    [-farmW - 0.5, 0, farmH + 0.5],
    [farmW + 0.5, 0, farmH + 0.5],
    [-farmW - 1, 0, 0],
    [farmW + 1, 0, 0],
  ], [farmW, farmH]);

  return (
    <div style={{
      width: '100%', height: 500, borderRadius: 12, overflow: 'hidden',
      border: '2px solid var(--semi-color-border)',
      background: 'linear-gradient(180deg, #bae6fd 0%, #e0f2fe 40%, #dcfce7 100%)',
    }}>
      <Canvas shadows dpr={[1, 2]} gl={{ antialias: true }}>
        <SceneSetup totalPlots={totalPlots} />
        <OrbitControls
          enablePan={true}
          enableZoom={true}
          enableRotate={true}
          minPolarAngle={Math.PI / 6}
          maxPolarAngle={Math.PI / 2.5}
          minDistance={4}
          maxDistance={25}
        />

        {/* Lighting */}
        <ambientLight intensity={0.6} />
        <directionalLight
          position={[8, 12, 8]}
          intensity={1.2}
          castShadow
          shadow-mapSize={[1024, 1024]}
          shadow-camera-near={0.5}
          shadow-camera-far={50}
          shadow-camera-left={-15}
          shadow-camera-right={15}
          shadow-camera-top={15}
          shadow-camera-bottom={-15}
        />
        <directionalLight position={[-5, 8, -5]} intensity={0.3} />
        <hemisphereLight args={['#87ceeb', '#4ade80', 0.3]} />

        {/* Sky color */}
        <color attach='background' args={['#e0f2fe']} />
        <fog attach='fog' args={['#e0f2fe', 20, 50]} />

        {/* Ground */}
        <Ground totalPlots={totalPlots} />

        {/* Fence */}
        <Fence totalPlots={totalPlots} />

        {/* Plots */}
        {plots.map((plot, i) => {
          const pos = getGridPos(i, totalPlots);
          return (
            <group key={plot.plot_index}>
              <SoilPlot
                position={pos}
                status={plot.status}
                onClick={() => handlePlotClick(plot.plot_index)}
              />

              {/* Crop based on status */}
              {plot.status === 0 && <EmptyPlotSign position={pos} />}
              {plot.status === 1 && (
                <GrowingCrop
                  position={pos}
                  progress={plot.progress}
                  cropType={plot.crop_name}
                  fertilized={plot.fertilized}
                />
              )}
              {plot.status === 2 && <MatureCrop position={pos} cropType={plot.crop_name} />}
              {plot.status === 3 && <EventCrop position={pos} eventType={plot.event_type} cropType={plot.crop_name} />}
              {plot.status === 4 && <WiltCrop position={pos} cropType={plot.crop_name} />}

              {/* Label on hover/select */}
              {selectedPlotIndex === plot.plot_index && (
                <PlotLabel
                  position={pos}
                  plot={plot}
                  t={t}
                  onAction={handlePlotAction}
                />
              )}
            </group>
          );
        })}

        {/* Dog */}
        {farmData?.dog && (
          <FarmDog dogData={farmData.dog} totalPlots={totalPlots} />
        )}

        {/* Decorative Trees */}
        {treePositions.map((pos, i) => (
          <DecoTree key={i} position={pos} scale={0.7 + (i % 3) * 0.2} />
        ))}
      </Canvas>
    </div>
  );
};

export default Farm3DView;
