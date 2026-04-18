import React, { useState, useEffect, useMemo, useRef, useCallback } from 'react';
import { Search, X } from 'lucide-react';
import { navGroups } from './Sidebar';
import { FEATURE_LEVEL_MAP } from '../constants';

const RECENT_KEY = 'farm_cmd_recent';
const MAX_RECENT = 6;

/**
 * Ctrl+K / Cmd+K 命令面板
 * - 键盘驱动：↑↓ 选择，Enter 跳转，Esc 关闭
 * - 空输入展示最近访问 + 所有页面
 * - 有输入时模糊匹配页面名/分组名/key
 * - 等级锁定项正常显示但禁用跳转
 */

const readRecent = () => {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(RECENT_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch (_) { return []; }
};

const writeRecent = (list) => {
  try { window.localStorage.setItem(RECENT_KEY, JSON.stringify(list)); } catch (_) {}
};

// 构建可搜索的扁平项列表（只构建一次）
const buildAllItems = () => {
  const items = [
    // 虚拟的 home（不属于任何 navGroup）
    {
      key: 'home',
      label: '主页',
      emoji: '\u{1F3E0}',
      groupKey: '__top',
      groupLabel: '导航',
      groupAccent: '#6dbb5c',
    },
  ];
  navGroups.forEach((group) => {
    group.items.forEach((item) => {
      items.push({
        ...item,
        groupKey: group.key,
        groupLabel: group.label,
        groupAccent: group.accent,
      });
    });
  });
  return items;
};

const ALL_ITEMS = buildAllItems();

const CommandPalette = ({ open, onClose, userLevel = 1, onNavigate, t }) => {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [recent, setRecent] = useState(readRecent);
  const inputRef = useRef(null);
  const listRef = useRef(null);

  // 打开 / 关闭时重置状态
  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedIndex(0);
      const id = setTimeout(() => inputRef.current?.focus(), 30);
      return () => clearTimeout(id);
    }
    return undefined;
  }, [open]);

  // 输入变化时重置选中到第 0 项
  useEffect(() => { setSelectedIndex(0); }, [query]);

  // 计算显示项
  const sections = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) {
      const recentItems = recent
        .map((k) => ALL_ITEMS.find((i) => i.key === k))
        .filter(Boolean);
      const recentKeys = new Set(recentItems.map((i) => i.key));
      const others = ALL_ITEMS.filter((i) => !recentKeys.has(i.key));
      const res = [];
      if (recentItems.length > 0) res.push({ title: '最近访问', items: recentItems });
      res.push({ title: '所有页面', items: others });
      return res;
    }
    // 简单模糊匹配：标签 / key / 分组名 里包含查询
    const matches = ALL_ITEMS.filter((i) => {
      const label = (t ? t(i.label) : i.label).toLowerCase();
      const groupLabel = (t ? t(i.groupLabel) : i.groupLabel).toLowerCase();
      return (
        label.includes(q) ||
        i.key.toLowerCase().includes(q) ||
        groupLabel.includes(q)
      );
    });
    return matches.length > 0 ? [{ title: '搜索结果', items: matches }] : [];
  }, [query, recent, t]);

  // 扁平化项目（用于键盘导航索引）
  const flatItems = useMemo(
    () => sections.flatMap((s) => s.items),
    [sections]
  );

  const handleSelect = useCallback((item) => {
    if (!item) return;
    const req = FEATURE_LEVEL_MAP[item.key];
    const locked = req && userLevel < req.level;
    if (locked) return;
    // 记录最近访问
    setRecent((prev) => {
      const next = [item.key, ...prev.filter((k) => k !== item.key)].slice(0, MAX_RECENT);
      writeRecent(next);
      return next;
    });
    onNavigate(item.key);
    onClose();
  }, [userLevel, onNavigate, onClose]);

  // 键盘导航
  useEffect(() => {
    if (!open) return undefined;
    const handler = (e) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex((i) => (flatItems.length > 0 ? (i + 1) % flatItems.length : 0));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex((i) => (flatItems.length > 0 ? (i - 1 + flatItems.length) % flatItems.length : 0));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        handleSelect(flatItems[selectedIndex]);
      } else if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, flatItems, selectedIndex, handleSelect, onClose]);

  // 确保选中项在视口内
  useEffect(() => {
    if (!listRef.current) return;
    const el = listRef.current.querySelector('.farm-cmd-item.selected');
    if (el) el.scrollIntoView({ block: 'nearest' });
  }, [selectedIndex]);

  if (!open) return null;

  let globalIdx = -1;

  return (
    <div className='farm-cmd-overlay' onMouseDown={onClose}>
      <div className='farm-cmd-panel' onMouseDown={(e) => e.stopPropagation()}>
        <div className='farm-cmd-searchbar'>
          <Search size={16} className='farm-cmd-search-icon' />
          <input
            ref={inputRef}
            type='text'
            className='farm-cmd-input'
            placeholder={t('搜索页面、功能...')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            spellCheck={false}
            autoComplete='off'
          />
          <button className='farm-cmd-close' onClick={onClose} aria-label='close'>
            <X size={14} />
          </button>
        </div>
        <div className='farm-cmd-list' ref={listRef}>
          {sections.length === 0 ? (
            <div className='farm-cmd-empty'>
              <div style={{ fontSize: 32, marginBottom: 8 }}>🔍</div>
              <div>{t('没有匹配的页面')}</div>
              <div style={{ fontSize: 12, color: 'var(--farm-text-3)', marginTop: 4 }}>
                {t('试试其他关键词')}
              </div>
            </div>
          ) : (
            sections.map((section) => (
              <div key={section.title}>
                <div className='farm-cmd-section-title'>{t(section.title)}</div>
                {section.items.map((item) => {
                  globalIdx += 1;
                  const req = FEATURE_LEVEL_MAP[item.key];
                  const locked = req && userLevel < req.level;
                  const isSelected = globalIdx === selectedIndex;
                  return (
                    <div
                      key={item.key}
                      className={`farm-cmd-item ${isSelected ? 'selected' : ''} ${locked ? 'locked' : ''}`}
                      style={{ '--item-accent': item.groupAccent }}
                      onMouseEnter={() => setSelectedIndex(globalIdx)}
                      onClick={() => handleSelect(item)}
                    >
                      <span className='farm-cmd-item-emoji'>{locked ? '🔒' : item.emoji}</span>
                      <span className='farm-cmd-item-label'>{t(item.label)}</span>
                      <span className='farm-cmd-item-group'>{t(item.groupLabel)}</span>
                      {locked && <span className='farm-cmd-item-lock'>Lv.{req.level}</span>}
                    </div>
                  );
                })}
              </div>
            ))
          )}
        </div>
        <div className='farm-cmd-footer'>
          <span className='farm-cmd-kbd-hint'>
            <kbd>↑</kbd><kbd>↓</kbd> {t('选择')}
          </span>
          <span className='farm-cmd-kbd-hint'>
            <kbd>Enter</kbd> {t('跳转')}
          </span>
          <span className='farm-cmd-kbd-hint'>
            <kbd>Esc</kbd> {t('关闭')}
          </span>
          <span className='farm-cmd-kbd-hint' style={{ marginLeft: 'auto' }}>
            <kbd>Ctrl</kbd>+<kbd>K</kbd>
          </span>
        </div>
      </div>
    </div>
  );
};

export default CommandPalette;
