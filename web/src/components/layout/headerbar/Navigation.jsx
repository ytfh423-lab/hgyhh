/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Home, LayoutDashboard, Store, BookOpen, Info } from 'lucide-react';
import SkeletonWrapper from '../components/SkeletonWrapper';

const NAV_ICONS = {
  home: Home,
  console: LayoutDashboard,
  pricing: Store,
  docs: BookOpen,
  about: Info,
};

const Navigation = ({
  mainNavLinks,
  isMobile,
  isLoading,
  userState,
  pricingRequireAuth,
}) => {
  const location = useLocation();

  const baseClasses =
    'group relative flex-shrink-0 flex items-center justify-center gap-1.5 font-semibold rounded-full transition-all duration-300 ease-out border backdrop-blur-sm';
  const spacingClasses = isMobile ? 'px-3 py-1.5 text-sm' : 'px-5 py-2 text-[14px]';
  const inactiveClasses =
    'border-transparent text-semi-color-text-1 hover:text-semi-color-primary hover:bg-semi-color-fill-1 hover:border-semi-color-primary/20 hover:shadow-sm';
  const activeClasses =
    'text-white border-transparent shadow-lg shadow-blue-500/30 bg-gradient-to-r from-blue-500 via-indigo-500 to-violet-500';

  const getTargetPath = (link) => {
    if (link.itemKey === 'console' && !userState.user) {
      return '/login';
    }
    if (link.itemKey === 'pricing' && pricingRequireAuth && !userState.user) {
      return '/login';
    }
    return link.to;
  };

  const isLinkActive = (link, targetPath) => {
    if (link.itemKey === 'home') {
      return location.pathname === '/';
    }
    if (link.itemKey === 'console') {
      return location.pathname.startsWith('/console');
    }
    return location.pathname === targetPath;
  };

  const renderNavLink = (link) => {
    const targetPath = getTargetPath(link);
    const isActive = isLinkActive(link, targetPath);
    const commonLinkClasses = `${baseClasses} ${spacingClasses} ${isActive ? activeClasses : inactiveClasses}`;
    const IconComponent = NAV_ICONS[link.icon];

    const linkContent = (
      <>
        {IconComponent && (
          <IconComponent
            size={15}
            className='relative z-[1]'
            style={{
              opacity: isActive ? 1 : 0.7,
              transition: 'opacity 0.2s ease',
            }}
          />
        )}
        <span className='relative z-[1]'>{link.text}</span>
        {!isActive && (
          <span className='absolute inset-0 rounded-full opacity-0 group-hover:opacity-100 transition-opacity duration-300 bg-gradient-to-r from-blue-500/8 to-violet-500/8' />
        )}
      </>
    );

    if (link.isExternal) {
      return (
        <a
          key={link.itemKey}
          href={link.externalLink}
          target='_blank'
          rel='noopener noreferrer'
          className={commonLinkClasses}
        >
          {linkContent}
        </a>
      );
    }

    return (
      <Link key={link.itemKey} to={targetPath} className={commonLinkClasses}>
        {linkContent}
      </Link>
    );
  };

  return (
    <nav className='flex flex-1 items-center gap-2 lg:gap-3 mx-2 md:mx-4 overflow-x-auto whitespace-nowrap scrollbar-hide'>
      <SkeletonWrapper
        loading={isLoading}
        type='navigation'
        count={4}
        width={60}
        height={16}
        isMobile={isMobile}
      >
        {mainNavLinks.map((link) => renderNavLink(link))}
      </SkeletonWrapper>
    </nav>
  );
};

export default Navigation;
