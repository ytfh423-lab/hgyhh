import React, { useRef, useMemo, useState, useEffect } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls';
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
  mature: '#eab308',
  wilt: '#a1a1aa',
  danger: '#ef4444',
  path: '#d6d3d1',
  pathDark: '#a8a29e',
  cloud: '#f0f9ff',
  waterDeep: '#0ea5e9',
  grassLight: '#86efac',
};

const CROP_COLORS = {
  watermelon: { body: '#22c55e', fruit: '#ef4444' },
  strawberry: { body: '#16a34a', fruit: '#ef4444' },
  carrot: { body: '#22c55e', fruit: '#f97316' },
  corn: { body: '#16a34a', fruit: '#eab308' },
  rice: { body: '#a3e635', fruit: '#fde047' },
  potato: { body: '#22c55e', fruit: '#a16207' },
  tomato: { body: '#16a34a', fruit: '#ef4444' },
  pumpkin: { body: '#16a34a', fruit: '#f97316' },
  default: { body: '#22c55e', fruit: '#84cc16', stem: '#166534' },
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

// ==================== OrbitControls (pure three.js) ====================
const CameraControls = () => {
  const { camera, gl } = useThree();
  const controlsRef = useRef();

  useEffect(() => {
    const controls = new OrbitControls(camera, gl.domElement);
    controls.enablePan = true;
    controls.enableZoom = true;
    controls.enableRotate = true;
    controls.enableDamping = true;
    controls.dampingFactor = 0.08;
    controls.rotateSpeed = 0.8;
    controls.zoomSpeed = 0.9;
    controls.panSpeed = 0.8;
    controls.minPolarAngle = Math.PI / 6;
    controls.maxPolarAngle = Math.PI / 2.5;
    controls.minDistance = 4;
    controls.maxDistance = 25;
    controlsRef.current = controls;
    return () => controls.dispose();
  }, [camera, gl]);

  useFrame(() => {
    controlsRef.current?.update();
  });

  return null;
};

// ==================== Ground ====================
const Ground = ({ totalPlots }) => {
  const cols = Math.min(COLS, totalPlots);
  const rows = Math.ceil(totalPlots / cols);
  const w = cols * (PLOT_SIZE + PLOT_GAP) + 3;
  const h = rows * (PLOT_SIZE + PLOT_GAP) + 3;

  const grassPatches = useMemo(() => {
    const patches = [];
    for (let i = 0; i < 24; i++) {
      patches.push({
        x: (Math.random() - 0.5) * (w + 4),
        z: (Math.random() - 0.5) * (h + 4),
        s: 0.25 + Math.random() * 0.45,
        r: Math.random() * Math.PI,
      });
    }
    return patches;
  }, [w, h]);

  return (
    <group>
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.05, 0]} receiveShadow>
        <planeGeometry args={[w + 4, h + 4]} />
        <meshStandardMaterial color={COLORS.grass} roughness={0.9} metalness={0} />
      </mesh>
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.06, 0]} receiveShadow>
        <planeGeometry args={[w + 6, h + 6]} />
        <meshStandardMaterial color={COLORS.grassDark} roughness={0.95} metalness={0} />
      </mesh>
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.045, h / 2 + 1.2]}>
        <planeGeometry args={[2.8, 1.8]} />
        <meshStandardMaterial color={COLORS.pathDark} roughness={1} metalness={0} />
      </mesh>
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.04, h / 2 + 1.2]}>
        <planeGeometry args={[2.5, 1.5]} />
        <meshStandardMaterial color={COLORS.path} roughness={1} metalness={0} />
      </mesh>
      {grassPatches.map((p, i) => (
        <mesh key={i} rotation={[-Math.PI / 2, p.r, 0]} position={[p.x, -0.04, p.z]}>
          <circleGeometry args={[p.s, 8]} />
          <meshStandardMaterial color={COLORS.grassLight} roughness={0.9} transparent opacity={0.35} />
        </mesh>
      ))}
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
      if (si === 2 && Math.abs(x) < 1.5) continue;
      posts.push([x, 0.3, z]);
    }
  });

  return (
    <group>
      {posts.map((pos, i) => (
        <group key={`post-${i}`} position={pos}>
          <mesh castShadow>
            <cylinderGeometry args={[0.06, 0.08, 0.7, 12]} />
            <meshStandardMaterial color={COLORS.fencePost} roughness={0.8} metalness={0.1} />
          </mesh>
          <mesh position={[0, 0.38, 0]}>
            <sphereGeometry args={[0.08, 12, 12]} />
            <meshStandardMaterial color={COLORS.fence} roughness={0.6} metalness={0.15} />
          </mesh>
        </group>
      ))}
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
              <cylinderGeometry args={[0.03, 0.03, dist, 8]} />
              <meshStandardMaterial color={COLORS.fence} roughness={0.6} metalness={0.15} />
            </mesh>
            <mesh position={[mx, 0.2, mz]} rotation={[0, angle, Math.PI / 2]}>
              <cylinderGeometry args={[0.03, 0.03, dist, 8]} />
              <meshStandardMaterial color={COLORS.fence} roughness={0.6} metalness={0.15} />
            </mesh>
          </group>
        );
      })}
    </group>
  );
};

// ==================== Water Ripple Effect ====================
const WaterRipple = ({ position }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      const t = state.clock.elapsedTime;
      const s = 1 + Math.sin(t * 3) * 0.15;
      ref.current.scale.set(s, s, 1);
      ref.current.material.opacity = 0.12 + Math.sin(t * 2) * 0.06;
    }
  });
  return (
    <mesh ref={ref} position={[position[0], PLOT_HEIGHT + 0.025, position[2]]} rotation={[-Math.PI / 2, 0, 0]}>
      <ringGeometry args={[PLOT_SIZE * 0.15, PLOT_SIZE * 0.42, 24]} />
      <meshStandardMaterial color={COLORS.waterDeep} transparent opacity={0.15} roughness={0.1} metalness={0.3} side={THREE.DoubleSide} />
    </mesh>
  );
};

// ==================== 3D Progress Ring ====================
const ProgressRing = ({ progress, position }) => {
  const ref = useRef();
  const pct = Math.max(0, Math.min(100, progress)) / 100;

  useFrame((state) => {
    if (ref.current) {
      ref.current.rotation.y = state.clock.elapsedTime * 0.5;
    }
  });

  return (
    <group ref={ref} position={position}>
      <mesh rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[0.18, 0.018, 8, 32]} />
        <meshStandardMaterial color='#e5e7eb' transparent opacity={0.35} />
      </mesh>
      <mesh rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[0.18, 0.025, 8, 32, Math.PI * 2 * pct]} />
        <meshStandardMaterial
          color={pct > 0.7 ? '#22c55e' : pct > 0.3 ? '#eab308' : '#38bdf8'}
          emissive={pct > 0.7 ? '#22c55e' : pct > 0.3 ? '#eab308' : '#38bdf8'}
          emissiveIntensity={0.4}
        />
      </mesh>
    </group>
  );
};

// ==================== Clouds ====================
const CloudMesh = ({ position, scale = 1 }) => {
  const ref = useRef();
  const speed = useMemo(() => 0.07 + Math.random() * 0.05, []);
  const offset = useMemo(() => Math.random() * Math.PI * 2, []);

  useFrame((state) => {
    if (ref.current) {
      ref.current.position.x = position[0] + Math.sin(state.clock.elapsedTime * speed + offset) * 2;
      ref.current.position.y = position[1] + Math.sin(state.clock.elapsedTime * speed * 0.5) * 0.3;
    }
  });

  return (
    <group ref={ref} position={position} scale={scale}>
      <mesh>
        <sphereGeometry args={[0.8, 16, 16]} />
        <meshStandardMaterial color={COLORS.cloud} roughness={1} transparent opacity={0.8} />
      </mesh>
      <mesh position={[0.7, -0.1, 0]}>
        <sphereGeometry args={[0.6, 16, 16]} />
        <meshStandardMaterial color={COLORS.cloud} roughness={1} transparent opacity={0.8} />
      </mesh>
      <mesh position={[-0.6, -0.1, 0.2]}>
        <sphereGeometry args={[0.55, 16, 16]} />
        <meshStandardMaterial color={COLORS.cloud} roughness={1} transparent opacity={0.8} />
      </mesh>
      <mesh position={[0.3, 0.25, -0.2]}>
        <sphereGeometry args={[0.5, 16, 16]} />
        <meshStandardMaterial color={COLORS.cloud} roughness={1} transparent opacity={0.8} />
      </mesh>
    </group>
  );
};

const Clouds = () => {
  const positions = useMemo(() => [
    [-6, 8, -4, 1.2],
    [5, 9, -6, 0.9],
    [8, 7.5, 3, 1.0],
    [-4, 8.5, 5, 0.8],
    [0, 9.5, -8, 1.1],
  ], []);

  return (
    <group>
      {positions.map(([x, y, z, s], i) => (
        <CloudMesh key={i} position={[x, y, z]} scale={s} />
      ))}
    </group>
  );
};

// ==================== Soil Plot ====================
const SOIL_LEVEL_COLORS = ['#92400e', '#854d0e', '#713f12', '#4a2c0a', '#2d1a06'];
const SOIL_LEVEL_BORDER = ['#d6d3d1', '#a3e635', '#22d3ee', '#a78bfa', '#f59e0b'];

const SoilPlot = ({ position, status, onClick, soilLevel = 1 }) => {
  const meshRef = useRef();
  const [hovered, setHovered] = useState(false);
  const scaleRef = useRef(1);
  const lvl = Math.max(1, Math.min(5, soilLevel)) - 1;

  const soilColor = useMemo(() => {
    if (status === 4) return COLORS.soilDry;
    if (status === 3) return COLORS.danger;
    return hovered ? COLORS.soilLight : SOIL_LEVEL_COLORS[lvl];
  }, [status, hovered, lvl]);

  useFrame(() => {
    if (meshRef.current) {
      const target = hovered ? 1.04 : 1;
      scaleRef.current += (target - scaleRef.current) * 0.12;
      meshRef.current.scale.set(scaleRef.current, 1, scaleRef.current);
    }
  });

  const handlePointerOver = (e) => {
    e.stopPropagation();
    setHovered(true);
    document.body.style.cursor = 'pointer';
  };

  const handlePointerOut = () => {
    setHovered(false);
    document.body.style.cursor = 'auto';
  };

  return (
    <group position={position}>
      <mesh
        ref={meshRef}
        position={[0, PLOT_HEIGHT / 2, 0]}
        castShadow receiveShadow
        onClick={(e) => { e.stopPropagation(); onClick(); }}
        onPointerOver={handlePointerOver}
        onPointerOut={handlePointerOut}
      >
        <boxGeometry args={[PLOT_SIZE, PLOT_HEIGHT, PLOT_SIZE]} />
        <meshStandardMaterial color={soilColor} roughness={1} metalness={0} />
      </mesh>
      {[-0.45, -0.15, 0.15, 0.45].map((offset, i) => (
        <mesh key={i} position={[0, PLOT_HEIGHT + 0.01, offset]} rotation={[-Math.PI / 2, 0, 0]}>
          <planeGeometry args={[PLOT_SIZE - 0.2, 0.06]} />
          <meshStandardMaterial color='#7c2d12' transparent opacity={0.3} />
        </mesh>
      ))}
      {status === 1 && (
        <mesh position={[0, PLOT_HEIGHT + 0.02, 0]} rotation={[-Math.PI / 2, 0, 0]}>
          <planeGeometry args={[PLOT_SIZE - 0.1, PLOT_SIZE - 0.1]} />
          <meshStandardMaterial color={COLORS.water} transparent opacity={0.12} roughness={0.1} metalness={0.2} />
        </mesh>
      )}
      {/* Soil level border glow */}
      {lvl > 0 && (
        <mesh position={[0, PLOT_HEIGHT + 0.005, 0]} rotation={[-Math.PI / 2, 0, 0]}>
          <ringGeometry args={[PLOT_SIZE * 0.48, PLOT_SIZE * 0.52, 4]} />
          <meshStandardMaterial
            color={SOIL_LEVEL_BORDER[lvl]}
            emissive={SOIL_LEVEL_BORDER[lvl]}
            emissiveIntensity={0.5}
            transparent opacity={0.6}
            side={THREE.DoubleSide}
          />
        </mesh>
      )}
      {/* Soil level indicator dots */}
      {lvl > 0 && Array.from({ length: lvl }, (_, i) => (
        <mesh key={`dot-${i}`} position={[
          -PLOT_SIZE * 0.35 + i * 0.18, PLOT_HEIGHT + 0.03, PLOT_SIZE * 0.42
        ]}>
          <sphereGeometry args={[0.04, 8, 8]} />
          <meshStandardMaterial
            color={SOIL_LEVEL_BORDER[lvl]}
            emissive={SOIL_LEVEL_BORDER[lvl]}
            emissiveIntensity={0.8}
          />
        </mesh>
      ))}
    </group>
  );
};

// ==================== Crop Models ====================
const EmptyPlotSign = ({ position }) => (
  <group position={[position[0], PLOT_HEIGHT + 0.01, position[2]]}>
    <mesh position={[0, 0.25, 0]}>
      <boxGeometry args={[0.03, 0.5, 0.03]} />
      <meshStandardMaterial color={COLORS.fencePost} roughness={0.8} />
    </mesh>
    <mesh position={[0, 0.5, 0]}>
      <boxGeometry args={[0.3, 0.2, 0.02]} />
      <meshStandardMaterial color='#fef3c7' roughness={0.5} />
    </mesh>
  </group>
);

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

  const stems = useMemo(() =>
    [[-0.2, -0.2], [0.2, -0.2], [0, 0.15], [-0.3, 0.1], [0.3, 0.1]].map(([ox, oz], i) => ({
      ox, oz, s: scale * (0.85 + i * 0.03), h: stemHeight * (0.7 + i * 0.08),
    })), [scale, stemHeight]);

  return (
    <group ref={groupRef} position={[position[0], PLOT_HEIGHT, position[2]]}>
      {stems.map(({ ox, oz, s, h }, i) => (
        <group key={i} position={[ox * scale, 0, oz * scale]}>
          <mesh position={[0, h / 2, 0]} castShadow>
            <cylinderGeometry args={[0.02 * s, 0.03 * s, h, 8]} />
            <meshStandardMaterial color={colors.stem || '#4d7c0f'} roughness={0.8} />
          </mesh>
          <mesh position={[0, h * 0.7, 0]} castShadow>
            <sphereGeometry args={[0.12 * s, 12, 12]} />
            <meshStandardMaterial color={colors.body} roughness={0.7} />
          </mesh>
          <mesh position={[0, h, 0]} castShadow>
            <coneGeometry args={[0.08 * s, 0.15 * s, 8]} />
            <meshStandardMaterial color={colors.body} roughness={0.7} />
          </mesh>
        </group>
      ))}
      <ProgressRing progress={progress} position={[0, stemHeight + 0.35, 0]} />
      {fertilized === 1 && <FertilizerEffect position={[0, stemHeight + 0.2, 0]} />}
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
      {[[-0.25, -0.2], [0.2, -0.25], [0, 0.2], [-0.3, 0.15], [0.25, 0.15]].map(([ox, oz], i) => (
        <group key={i} position={[ox, 0, oz]}>
          <mesh position={[0, 0.3, 0]} castShadow>
            <cylinderGeometry args={[0.025, 0.04, 0.6, 10]} />
            <meshStandardMaterial color='#4d7c0f' roughness={0.8} />
          </mesh>
          <mesh position={[0.08, 0.35, 0]} rotation={[0, 0, 0.5]} castShadow>
            <boxGeometry args={[0.18, 0.04, 0.08]} />
            <meshStandardMaterial color={colors.body} roughness={0.7} />
          </mesh>
          <mesh position={[-0.08, 0.28, 0]} rotation={[0, 0, -0.5]} castShadow>
            <boxGeometry args={[0.18, 0.04, 0.08]} />
            <meshStandardMaterial color={colors.body} roughness={0.7} />
          </mesh>
          <mesh position={[0, 0.55, 0]} castShadow>
            <sphereGeometry args={[0.12, 16, 16]} />
            <meshStandardMaterial color={colors.fruit} roughness={0.5} metalness={0.05} />
          </mesh>
        </group>
      ))}
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
          <octahedronGeometry args={[0.03, 1]} />
          <meshStandardMaterial color={COLORS.mature} emissive={COLORS.mature} emissiveIntensity={1.2} />
        </mesh>
      ))}
    </group>
  );
};

const EventCrop = ({ position, eventType }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) {
      ref.current.position.x = position[0] + Math.sin(state.clock.elapsedTime * 8) * 0.02;
    }
  });
  const isDrought = eventType === 'drought';
  return (
    <group ref={ref} position={[position[0], PLOT_HEIGHT, position[2]]}>
      {[[-0.2, -0.15], [0.15, -0.2], [0, 0.15]].map(([ox, oz], i) => (
        <group key={i} position={[ox, 0, oz]} rotation={[0.3, 0, i * 0.5]}>
          <mesh position={[0, 0.2, 0]} castShadow>
            <cylinderGeometry args={[0.02, 0.03, 0.4, 8]} />
            <meshStandardMaterial color={isDrought ? '#a16207' : '#4d7c0f'} roughness={0.8} />
          </mesh>
          <mesh position={[0, 0.35, 0]} castShadow>
            <sphereGeometry args={[0.08, 10, 10]} />
            <meshStandardMaterial color={isDrought ? '#d97706' : '#84cc16'} roughness={0.7} />
          </mesh>
        </group>
      ))}
      {isDrought ? <DroughtEffect /> : <BugEffect />}
    </group>
  );
};

const DroughtEffect = () => {
  const ref = useRef();
  useFrame((state) => { if (ref.current) ref.current.rotation.y = state.clock.elapsedTime * 0.5; });
  return (
    <group ref={ref} position={[0, 0.6, 0]}>
      {[0, 1, 2].map(i => (
        <mesh key={i} position={[Math.cos(i * Math.PI * 2 / 3) * 0.2, i * 0.1, Math.sin(i * Math.PI * 2 / 3) * 0.2]}>
          <torusGeometry args={[0.08, 0.02, 8, 16]} />
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
          <sphereGeometry args={[0.03, 8, 8]} />
          <meshStandardMaterial color='#1a1a1a' />
        </mesh>
      ))}
    </group>
  );
};

const WiltCrop = ({ position }) => (
  <group position={[position[0], PLOT_HEIGHT, position[2]]}>
    {[[-0.2, -0.15], [0.15, -0.2], [0, 0.15]].map(([ox, oz], i) => (
      <group key={i} position={[ox, 0, oz]} rotation={[0.6, 0, i * 0.8]}>
        <mesh position={[0, 0.12, 0]} castShadow>
          <cylinderGeometry args={[0.015, 0.025, 0.25, 8]} />
          <meshStandardMaterial color='#78716c' roughness={0.9} />
        </mesh>
        <mesh position={[0, 0.22, 0]} castShadow>
          <sphereGeometry args={[0.06, 10, 10]} />
          <meshStandardMaterial color={COLORS.wilt} roughness={0.8} />
        </mesh>
      </group>
    ))}
    <WarningSign position={[0, 0.7, 0]} />
  </group>
);

const WarningSign = ({ position }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) ref.current.position.y = position[1] + Math.sin(state.clock.elapsedTime * 3) * 0.05;
  });
  return (
    <group ref={ref} position={position}>
      <mesh>
        <coneGeometry args={[0.08, 0.12, 6]} />
        <meshStandardMaterial color={COLORS.danger} emissive={COLORS.danger} emissiveIntensity={0.5} />
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
        <mesh position={[0, 0.25, 0]} castShadow>
          <boxGeometry args={[0.25, 0.2, 0.4]} />
          <meshStandardMaterial color={isAdult ? '#92400e' : '#d97706'} />
        </mesh>
        <mesh position={[0, 0.35, 0.22]} castShadow>
          <boxGeometry args={[0.2, 0.18, 0.2]} />
          <meshStandardMaterial color={isAdult ? '#a16207' : '#eab308'} />
        </mesh>
        <mesh position={[0, 0.32, 0.35]} castShadow>
          <boxGeometry args={[0.1, 0.08, 0.1]} />
          <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
        </mesh>
        <mesh position={[0, 0.33, 0.41]}>
          <sphereGeometry args={[0.03, 10, 10]} />
          <meshStandardMaterial color='#1c1917' metalness={0.3} roughness={0.2} />
        </mesh>
        <mesh position={[-0.06, 0.39, 0.32]}>
          <sphereGeometry args={[0.025, 10, 10]} />
          <meshStandardMaterial color='#1c1917' metalness={0.3} roughness={0.2} />
        </mesh>
        <mesh position={[0.06, 0.39, 0.32]}>
          <sphereGeometry args={[0.025, 10, 10]} />
          <meshStandardMaterial color='#1c1917' metalness={0.3} roughness={0.2} />
        </mesh>
        <mesh position={[-0.1, 0.44, 0.2]} rotation={[0, 0, -0.3]}>
          <boxGeometry args={[0.08, 0.12, 0.06]} />
          <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
        </mesh>
        <mesh position={[0.1, 0.44, 0.2]} rotation={[0, 0, 0.3]}>
          <boxGeometry args={[0.08, 0.12, 0.06]} />
          <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
        </mesh>
        {[[-0.08, -0.15], [0.08, -0.15], [-0.08, 0.15], [0.08, 0.15]].map(([x, z], i) => (
          <mesh key={i} position={[x, 0.08, z]} castShadow>
            <boxGeometry args={[0.06, 0.16, 0.06]} />
            <meshStandardMaterial color={isAdult ? '#78350f' : '#a16207'} />
          </mesh>
        ))}
        <group ref={tailRef} position={[0, 0.35, -0.2]}>
          <mesh position={[0, 0.08, -0.05]} rotation={[0.5, 0, 0]}>
            <cylinderGeometry args={[0.025, 0.015, 0.18, 8]} />
            <meshStandardMaterial color={isAdult ? '#a16207' : '#eab308'} roughness={0.7} />
          </mesh>
        </group>
      </group>
    </group>
  );
};

// ==================== Decorative Trees ====================
const DecoTree = ({ position, scale = 1 }) => {
  const ref = useRef();
  useFrame((state) => {
    if (ref.current) ref.current.rotation.y = Math.sin(state.clock.elapsedTime * 0.5 + position[0]) * 0.05;
  });
  return (
    <group ref={ref} position={position} scale={scale}>
      <mesh position={[0, 0.35, 0]} castShadow>
        <cylinderGeometry args={[0.06, 0.1, 0.7, 12]} />
        <meshStandardMaterial color={COLORS.trunk} roughness={0.9} metalness={0} />
      </mesh>
      <mesh position={[0, 0.75, 0]} castShadow>
        <coneGeometry args={[0.4, 0.5, 16]} />
        <meshStandardMaterial color={COLORS.leaves} roughness={0.8} />
      </mesh>
      <mesh position={[0, 1.0, 0]} castShadow>
        <coneGeometry args={[0.3, 0.4, 16]} />
        <meshStandardMaterial color={COLORS.leavesDark} roughness={0.8} />
      </mesh>
      <mesh position={[0, 1.2, 0]} castShadow>
        <coneGeometry args={[0.2, 0.35, 16]} />
        <meshStandardMaterial color={COLORS.leaves} roughness={0.8} />
      </mesh>
    </group>
  );
};

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

// ==================== Plot Info Overlay (pure DOM) ====================
const PlotInfoOverlay = ({ plot, t, onAction, onClose, farmData }) => {
  if (!plot) return null;
  const statusText = {
    0: '空地',
    1: '生长中',
    2: '已成熟 ✨',
    3: plot.event_type === 'drought' ? '干旱!' : '虫害!',
    4: '枯萎!',
  };
  const borderColor = plot.status === 2 ? '#22c55e' : plot.status >= 3 ? '#ef4444' : '#e5e7eb';
  const statusBg = plot.status === 2 ? 'rgba(34,197,94,0.08)' : plot.status >= 3 ? 'rgba(239,68,68,0.08)' : 'transparent';
  const btnSt = (bg) => ({
    background: bg, border: 'none', borderRadius: 6, padding: '5px 12px',
    fontSize: 12, cursor: 'pointer', color: 'white', fontWeight: 600,
    transition: 'transform 0.1s, opacity 0.1s', boxShadow: `0 2px 8px ${bg}44`,
  });

  return (
    <div style={{
      position: 'absolute', top: 12, right: 12, zIndex: 10,
      background: 'rgba(255,255,255,0.96)', borderRadius: 12, padding: '12px 16px',
      boxShadow: '0 8px 32px rgba(0,0,0,0.12), 0 2px 8px rgba(0,0,0,0.08)', minWidth: 180,
      border: `2px solid ${borderColor}`, backdropFilter: 'blur(12px)',
      animation: 'fadeSlideIn 0.2s ease-out',
    }}>
      <style>{`@keyframes fadeSlideIn { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: translateY(0); } }`}</style>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <span style={{ fontWeight: 700, fontSize: 14 }}>
          {plot.crop_emoji || '📍'} {plot.plot_index + 1}{t('号地')}
        </span>
        <button onClick={onClose} style={{
          background: 'none', border: 'none', cursor: 'pointer', fontSize: 16, color: '#9ca3af', padding: '0 2px',
          transition: 'color 0.15s',
        }} onMouseOver={(e) => e.target.style.color = '#6b7280'} onMouseOut={(e) => e.target.style.color = '#9ca3af'}>✕</button>
      </div>
      <div style={{
        fontSize: 12, fontWeight: 600, marginBottom: 8, padding: '4px 8px', borderRadius: 6,
        background: statusBg,
        color: plot.status === 2 ? '#16a34a' : plot.status >= 3 ? '#dc2626' : '#6b7280',
      }}>
        {plot.crop_name ? `${plot.crop_name} · ` : ''}{statusText[plot.status] || ''}
      </div>
      <div style={{ fontSize: 11, color: '#78716c', marginBottom: 8, display: 'flex', alignItems: 'center', gap: 6 }}>
        <span>🌱 {t('泥土')} Lv.{plot.soil_level || 1}</span>
        {(plot.soil_level || 1) > 1 && <span style={{ color: '#16a34a' }}>⚡ -{((plot.soil_level || 1) - 1) * (farmData?.soil_speed_bonus || 10)}%</span>}
        {(plot.soil_level || 1) < (farmData?.soil_max_level || 5) && (
          <span style={{ color: '#a78bfa' }}>→ Lv.{(plot.soil_level || 1) + 1}: ${farmData?.soil_upgrade_prices?.[String((plot.soil_level || 1) + 1)]?.toFixed(2) || '?'}</span>
        )}
      </div>
      {plot.status === 1 && (
        <div style={{ marginBottom: 8 }}>
          <div style={{ background: '#e5e7eb', borderRadius: 6, height: 8, overflow: 'hidden' }}>
            <div style={{
              background: 'linear-gradient(90deg, #22c55e, #84cc16)',
              height: '100%', width: `${plot.progress}%`, borderRadius: 6,
              transition: 'width 0.3s ease',
            }} />
          </div>
          <div style={{ fontSize: 11, color: '#9ca3af', marginTop: 3, fontWeight: 500 }}>{plot.progress}%</div>
        </div>
      )}
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {plot.status === 1 && (
          <>
            <button onClick={() => onAction('water', plot.plot_index)} style={btnSt('#38bdf8')}>💧 {t('浇水')}</button>
            {plot.fertilized === 0 && (
              <button onClick={() => onAction('fertilize', plot.plot_index)} style={btnSt('#06b6d4')}>🧴 {t('施肥')}</button>
            )}
          </>
        )}
        {plot.status === 3 && plot.event_type === 'drought' && (
          <button onClick={() => onAction('water', plot.plot_index)} style={btnSt('#ef4444')}>💧 {t('浇水')}</button>
        )}
        {plot.status === 3 && plot.event_type !== 'drought' && (
          <button onClick={() => onAction('treat', plot.plot_index)} style={btnSt('#f59e0b')}>💊 {t('治疗')}</button>
        )}
        {plot.status === 4 && (
          <button onClick={() => onAction('water', plot.plot_index)} style={btnSt('#ef4444')}>💧 {t('浇水')}</button>
        )}
        {(plot.soil_level || 1) < (farmData?.soil_max_level || 5) && (
          <button onClick={() => onAction('upgrade-soil', plot.plot_index)} style={btnSt('#8b5cf6')}>⬆️ {t('升级泥土')}</button>
        )}
      </div>
    </div>
  );
};

// ==================== Main 3D Farm Component ====================
const Farm3DView = ({ farmData, doAction, t, selectedPlotIndex, setSelectedPlotIndex }) => {
  const plots = farmData?.plots || [];
  const totalPlots = plots.length;

  const handlePlotAction = (action, plotIndex) => {
    if (action === 'water') doAction('/api/farm/water', { plot_index: plotIndex });
    else if (action === 'fertilize') doAction('/api/farm/fertilize', { plot_index: plotIndex });
    else if (action === 'treat') doAction('/api/farm/treat', { plot_index: plotIndex });
    else if (action === 'upgrade-soil') doAction('/api/farm/upgrade-soil', { plot_index: plotIndex });
  };

  const handlePlotClick = (plotIndex) => {
    setSelectedPlotIndex(selectedPlotIndex === plotIndex ? null : plotIndex);
  };

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

  const selectedPlot = plots.find(p => p.plot_index === selectedPlotIndex);

  return (
    <div style={{
      width: '100%', height: 500, borderRadius: 12, overflow: 'hidden',
      border: '2px solid var(--semi-color-border)',
      background: 'linear-gradient(180deg, #bae6fd 0%, #e0f2fe 40%, #dcfce7 100%)',
      position: 'relative',
    }}>
      {/* DOM overlay for selected plot info */}
      <PlotInfoOverlay
        plot={selectedPlot}
        t={t}
        onAction={handlePlotAction}
        farmData={farmData}
        onClose={() => setSelectedPlotIndex(null)}
      />

      <Canvas shadows dpr={[1, 2]} gl={{ antialias: true, toneMapping: THREE.ACESFilmicToneMapping, toneMappingExposure: 1.1 }}>
        <SceneSetup totalPlots={totalPlots} />
        <CameraControls />

        <ambientLight intensity={0.5} />
        <directionalLight
          position={[8, 12, 8]} intensity={1.3} castShadow
          shadow-mapSize={[2048, 2048]}
          shadow-bias={-0.0005}
          shadow-camera-near={0.5} shadow-camera-far={50}
          shadow-camera-left={-15} shadow-camera-right={15}
          shadow-camera-top={15} shadow-camera-bottom={-15}
        />
        <directionalLight position={[-5, 8, -5]} intensity={0.35} />
        <pointLight position={[0, 6, 0]} intensity={0.15} color='#fef3c7' />
        <hemisphereLight args={['#87ceeb', '#4ade80', 0.35]} />

        <color attach='background' args={['#e0f2fe']} />
        <fog attach='fog' args={['#e0f2fe', 18, 45]} />

        <Ground totalPlots={totalPlots} />
        <Fence totalPlots={totalPlots} />

        {plots.map((plot, i) => {
          const pos = getGridPos(i, totalPlots);
          return (
            <group key={plot.plot_index}>
              <SoilPlot position={pos} status={plot.status} soilLevel={plot.soil_level} onClick={() => handlePlotClick(plot.plot_index)} />
              {plot.status === 0 && <EmptyPlotSign position={pos} />}
              {plot.status === 1 && (
                <GrowingCrop position={pos} progress={plot.progress} cropType={plot.crop_name} fertilized={plot.fertilized} />
              )}
              {plot.status === 2 && <MatureCrop position={pos} cropType={plot.crop_name} />}
              {plot.status === 3 && <EventCrop position={pos} eventType={plot.event_type} />}
              {plot.status === 4 && <WiltCrop position={pos} />}
              {plot.status === 1 && <WaterRipple position={pos} />}
            </group>
          );
        })}

        {farmData?.dog && <FarmDog dogData={farmData.dog} totalPlots={totalPlots} />}

        {treePositions.map((pos, i) => (
          <DecoTree key={i} position={pos} scale={0.7 + (i % 3) * 0.2} />
        ))}

        <Clouds />
      </Canvas>
    </div>
  );
};

export default Farm3DView;
