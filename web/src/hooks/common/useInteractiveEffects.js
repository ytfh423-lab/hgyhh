import { useEffect, useRef, useState, useCallback } from 'react';

/**
 * Scroll-triggered reveal animation using IntersectionObserver.
 * Returns a ref to attach to the element you want to animate.
 * @param {Object} options
 * @param {number} options.threshold - visibility threshold (0-1)
 * @param {string} options.rootMargin - margin around root
 * @param {boolean} options.once - only trigger once (default true)
 */
export function useScrollReveal({
  threshold = 0.15,
  rootMargin = '0px 0px -60px 0px',
  once = true,
} = {}) {
  const ref = useRef(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          if (once) observer.unobserve(el);
        } else if (!once) {
          setIsVisible(false);
        }
      },
      { threshold, rootMargin }
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, [threshold, rootMargin, once]);

  return [ref, isVisible];
}

/**
 * Mouse-following glow effect for a container.
 * Returns a ref and the CSS custom properties for the glow position.
 */
export function useMouseGlow() {
  const ref = useRef(null);
  const [glowPos, setGlowPos] = useState({ x: 50, y: 50 });
  const rafRef = useRef(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const handleMouseMove = (e) => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
      rafRef.current = requestAnimationFrame(() => {
        const rect = el.getBoundingClientRect();
        const x = ((e.clientX - rect.left) / rect.width) * 100;
        const y = ((e.clientY - rect.top) / rect.height) * 100;
        setGlowPos({ x, y });
      });
    };

    el.addEventListener('mousemove', handleMouseMove);
    return () => {
      el.removeEventListener('mousemove', handleMouseMove);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, []);

  const glowStyle = {
    '--glow-x': `${glowPos.x}%`,
    '--glow-y': `${glowPos.y}%`,
  };

  return [ref, glowStyle];
}

/**
 * 3D tilt effect on hover for cards.
 * Returns a ref to attach to the card element.
 * @param {Object} options
 * @param {number} options.maxTilt - max tilt in degrees (default 8)
 * @param {number} options.scale - scale on hover (default 1.02)
 * @param {number} options.speed - transition speed in ms (default 400)
 */
export function useTiltEffect({ maxTilt = 8, scale = 1.02, speed = 400 } = {}) {
  const ref = useRef(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    el.style.transition = `transform ${speed}ms cubic-bezier(0.03, 0.98, 0.52, 0.99)`;
    el.style.willChange = 'transform';

    const handleMouseMove = (e) => {
      const rect = el.getBoundingClientRect();
      const centerX = rect.left + rect.width / 2;
      const centerY = rect.top + rect.height / 2;
      const rotateX = ((e.clientY - centerY) / (rect.height / 2)) * -maxTilt;
      const rotateY = ((e.clientX - centerX) / (rect.width / 2)) * maxTilt;
      el.style.transform = `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale3d(${scale}, ${scale}, ${scale})`;
    };

    const handleMouseLeave = () => {
      el.style.transform =
        'perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)';
    };

    el.addEventListener('mousemove', handleMouseMove);
    el.addEventListener('mouseleave', handleMouseLeave);
    return () => {
      el.removeEventListener('mousemove', handleMouseMove);
      el.removeEventListener('mouseleave', handleMouseLeave);
    };
  }, [maxTilt, scale, speed]);

  return ref;
}

/**
 * Animated counter that counts up from 0 to target.
 * @param {number} target - target number
 * @param {number} duration - animation duration in ms (default 2000)
 * @param {boolean} start - whether to start the animation
 */
export function useCountUp(target, duration = 2000, start = false) {
  const [count, setCount] = useState(0);
  const rafRef = useRef(null);

  useEffect(() => {
    if (!start) {
      setCount(0);
      return;
    }

    let startTime = null;
    const animate = (timestamp) => {
      if (!startTime) startTime = timestamp;
      const progress = Math.min((timestamp - startTime) / duration, 1);
      const eased = 1 - Math.pow(1 - progress, 3); // ease-out cubic
      setCount(Math.round(eased * target));
      if (progress < 1) {
        rafRef.current = requestAnimationFrame(animate);
      }
    };

    rafRef.current = requestAnimationFrame(animate);
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [target, duration, start]);

  return count;
}

/**
 * Staggered children reveal - returns visibility state for N items.
 * @param {number} count - number of items
 * @param {number} staggerDelay - delay between each item in ms (default 60)
 * @param {boolean} trigger - when true, starts the stagger animation
 */
export function useStaggerReveal(count, staggerDelay = 60, trigger = false) {
  const [visibleItems, setVisibleItems] = useState(new Set());

  useEffect(() => {
    if (!trigger) {
      setVisibleItems(new Set());
      return;
    }

    const timers = [];
    for (let i = 0; i < count; i++) {
      const timer = setTimeout(() => {
        setVisibleItems((prev) => new Set([...prev, i]));
      }, i * staggerDelay);
      timers.push(timer);
    }
    return () => timers.forEach(clearTimeout);
  }, [count, staggerDelay, trigger]);

  return visibleItems;
}

/**
 * Magnetic hover effect - element slightly follows cursor on hover.
 * Returns a ref and style to apply.
 * @param {number} strength - movement strength in px (default 10)
 */
export function useMagneticHover(strength = 10) {
  const ref = useRef(null);
  const [offset, setOffset] = useState({ x: 0, y: 0 });

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const handleMouseMove = (e) => {
      const rect = el.getBoundingClientRect();
      const centerX = rect.left + rect.width / 2;
      const centerY = rect.top + rect.height / 2;
      const dx = (e.clientX - centerX) / (rect.width / 2);
      const dy = (e.clientY - centerY) / (rect.height / 2);
      setOffset({ x: dx * strength, y: dy * strength });
    };

    const handleMouseLeave = () => {
      setOffset({ x: 0, y: 0 });
    };

    el.addEventListener('mousemove', handleMouseMove);
    el.addEventListener('mouseleave', handleMouseLeave);
    return () => {
      el.removeEventListener('mousemove', handleMouseMove);
      el.removeEventListener('mouseleave', handleMouseLeave);
    };
  }, [strength]);

  const style = {
    transform: `translate(${offset.x}px, ${offset.y}px)`,
    transition: offset.x === 0 && offset.y === 0
      ? 'transform 0.4s cubic-bezier(0.25, 0.46, 0.45, 0.94)'
      : 'transform 0.15s ease-out',
  };

  return [ref, style];
}
