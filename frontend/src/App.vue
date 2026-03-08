<template>
  <router-view />
</template>

<script setup lang="ts">
import { useThemeStore } from './stores/theme'

// Initialize theme store to apply theme on app load
const themeStore = useThemeStore()
</script>

<style>
:root {
  /* Light theme (default) */
  --bg-primary: #ffffff;
  --bg-secondary: #f5f7f9;
  --bg-tertiary: #ebeef5;
  --bg-hover: rgba(0, 0, 0, 0.05);
  
  --text-primary: #1a1a1a;
  --text-secondary: #606266;
  --text-tertiary: #909399;
  --text-placeholder: #c0c4cc;
  
  --border-primary: #e4e7ed;
  --border-secondary: #dcdfe6;
  
  --shadow-sm: 0 2px 4px rgba(0, 0, 0, 0.08);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.12);
  
  --accent-primary: #409eff;
  --accent-success: #67c23a;
  --accent-warning: #e6a23c;
  --accent-danger: #f56c6c;
  
  --sidebar-bg: #ffffff;
  --card-bg: #ffffff;
  --input-bg: #ffffff;
  --code-bg: #f8f9fa;
}

[data-theme="dark"] {
  /* Modern Dark Theme - Clean neutral grays */
  --bg-primary: #1e1e1e;
  --bg-secondary: #252526;
  --bg-tertiary: #2d2d2d;
  --bg-hover: rgba(255, 255, 255, 0.06);
  
  --text-primary: #e6e6e6;
  --text-secondary: #9d9d9d;
  --text-tertiary: #6e6e6e;
  --text-placeholder: #4a4a4a;
  
  --border-primary: #3c3c3c;
  --border-secondary: #4a4a4a;
  
  --shadow-sm: 0 2px 4px rgba(0, 0, 0, 0.4);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.5);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.6);
  
  --accent-primary: #0078d4;
  --accent-success: #3fb950;
  --accent-warning: #d29922;
  --accent-danger: #f85149;
  
  --sidebar-bg: #1e1e1e;
  --card-bg: #252526;
  --input-bg: #3c3c3c;
  --code-bg: #1e1e1e;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen,
    Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  transition: background-color 0.3s ease, color 0.3s ease;
}

/* Global transitions for smooth theme switching */
.theme-transition,
.theme-transition *,
.theme-transition *::before,
.theme-transition *::after {
  transition: background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease, box-shadow 0.3s ease !important;
}

/* Scrollbar styling */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-tertiary);
}

::-webkit-scrollbar-thumb {
  background: var(--text-tertiary);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

/* Dark mode Element Plus overrides */

/* Default button */
[data-theme="dark"] .el-button--default {
  background-color: var(--bg-tertiary);
  border-color: var(--border-primary);
  color: var(--text-primary);
}

[data-theme="dark"] .el-button--default:hover,
[data-theme="dark"] .el-button--default:focus {
  background-color: #404040;
  border-color: #505050;
  color: var(--text-primary);
}

/* Primary button */
[data-theme="dark"] .el-button--primary {
  background-color: var(--accent-primary);
  border-color: var(--accent-primary);
}

[data-theme="dark"] .el-button--primary:hover,
[data-theme="dark"] .el-button--primary:focus {
  background-color: #1a8ceb;
  border-color: #1a8ceb;
}

/* Success button */
[data-theme="dark"] .el-button--success {
  background-color: var(--accent-success);
  border-color: var(--accent-success);
}

[data-theme="dark"] .el-button--success:hover,
[data-theme="dark"] .el-button--success:focus {
  background-color: #4cc764;
  border-color: #4cc764;
}

/* Warning button */
[data-theme="dark"] .el-button--warning {
  background-color: var(--accent-warning);
  border-color: var(--accent-warning);
}

[data-theme="dark"] .el-button--warning:hover,
[data-theme="dark"] .el-button--warning:focus {
  background-color: #e5a92e;
  border-color: #e5a92e;
}

/* Danger button */
[data-theme="dark"] .el-button--danger {
  background-color: var(--accent-danger);
  border-color: var(--accent-danger);
}

[data-theme="dark"] .el-button--danger:hover,
[data-theme="dark"] .el-button--danger:focus {
  background-color: #f96b64;
  border-color: #f96b64;
}

/* Info button */
[data-theme="dark"] .el-button--info {
  background-color: #505050;
  border-color: #505050;
  color: var(--text-primary);
}

[data-theme="dark"] .el-button--info:hover,
[data-theme="dark"] .el-button--info:focus {
  background-color: #606060;
  border-color: #606060;
}

/* Text button */
[data-theme="dark"] .el-button--text {
  color: var(--accent-primary);
}

[data-theme="dark"] .el-button--text:hover,
[data-theme="dark"] .el-button--text:focus {
  color: #1a8ceb;
}

/* Link button */
[data-theme="dark"] .el-button.is-link {
  color: var(--accent-primary);
}

[data-theme="dark"] .el-button.is-link:hover {
  color: #1a8ceb;
}

/* Plain buttons */
[data-theme="dark"] .el-button--primary.is-plain {
  background-color: rgba(0, 120, 212, 0.15);
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}

[data-theme="dark"] .el-button--primary.is-plain:hover {
  background-color: var(--accent-primary);
  color: white;
}

[data-theme="dark"] .el-button--success.is-plain {
  background-color: rgba(63, 185, 80, 0.15);
  border-color: var(--accent-success);
  color: var(--accent-success);
}

[data-theme="dark"] .el-button--success.is-plain:hover {
  background-color: var(--accent-success);
  color: white;
}

[data-theme="dark"] .el-button--warning.is-plain {
  background-color: rgba(210, 153, 34, 0.15);
  border-color: var(--accent-warning);
  color: var(--accent-warning);
}

[data-theme="dark"] .el-button--warning.is-plain:hover {
  background-color: var(--accent-warning);
  color: white;
}

[data-theme="dark"] .el-button--danger.is-plain {
  background-color: rgba(248, 81, 73, 0.15);
  border-color: var(--accent-danger);
  color: var(--accent-danger);
}

[data-theme="dark"] .el-button--danger.is-plain:hover {
  background-color: var(--accent-danger);
  color: white;
}

[data-theme="dark"] .el-button--info.is-plain {
  background-color: rgba(80, 80, 80, 0.15);
  border-color: #606060;
  color: var(--text-secondary);
}

[data-theme="dark"] .el-button--info.is-plain:hover {
  background-color: #505050;
  color: white;
}

/* Disabled buttons */
[data-theme="dark"] .el-button.is-disabled,
[data-theme="dark"] .el-button.is-disabled:hover {
  background-color: var(--bg-tertiary);
  border-color: var(--border-primary);
  color: var(--text-placeholder);
  cursor: not-allowed;
}

[data-theme="dark"] .el-input__wrapper {
  background-color: var(--input-bg);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

[data-theme="dark"] .el-input__inner {
  color: var(--text-primary);
}

[data-theme="dark"] .el-input__inner::placeholder {
  color: var(--text-placeholder);
}

[data-theme="dark"] .el-select__wrapper {
  background-color: var(--input-bg);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

[data-theme="dark"] .el-select__placeholder {
  color: var(--text-placeholder);
}

[data-theme="dark"] .el-select__selected-item {
  color: var(--text-primary);
}

[data-theme="dark"] .el-select-dropdown {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-select-dropdown__item {
  color: var(--text-primary);
}

[data-theme="dark"] .el-select-dropdown__item:hover {
  background-color: var(--bg-hover);
  color: var(--text-primary);
}

[data-theme="dark"] .el-select-dropdown__item.is-selected {
  color: var(--accent-primary);
  font-weight: bold;
}

[data-theme="dark"] .el-select-dropdown__item.is-disabled {
  color: var(--text-placeholder);
}

/* Popper / Dropdown menus */
[data-theme="dark"] .el-popper {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-popper.is-light {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-popper.is-light .el-popper__arrow::before {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-dropdown-menu {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-dropdown-menu__item {
  color: var(--text-primary);
}

[data-theme="dark"] .el-dropdown-menu__item:hover,
[data-theme="dark"] .el-dropdown-menu__item:focus {
  background-color: var(--bg-hover);
  color: var(--text-primary);
}

[data-theme="dark"] .el-dropdown-menu__item--divided {
  border-top-color: var(--border-primary);
}

[data-theme="dark"] .el-dropdown-menu__item.is-disabled {
  color: var(--text-placeholder);
}

/* Cascader dropdown */
[data-theme="dark"] .el-cascader__dropdown {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-cascader-menu {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-cascader-node {
  color: var(--text-primary);
}

[data-theme="dark"] .el-cascader-node:hover {
  background-color: var(--bg-hover);
}

[data-theme="dark"] .el-cascader-node.is-active {
  color: var(--accent-primary);
}

/* Autocomplete dropdown */
[data-theme="dark"] .el-autocomplete-suggestion {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-autocomplete-suggestion li {
  color: var(--text-primary);
}

[data-theme="dark"] .el-autocomplete-suggestion li:hover {
  background-color: var(--bg-hover);
}

/* Popover */
[data-theme="dark"] .el-popover {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
  color: var(--text-primary);
}

[data-theme="dark"] .el-popover__title {
  color: var(--text-primary);
}

/* Tooltip */
[data-theme="dark"] .el-tooltip__popper {
  background-color: #4a4a4a;
  color: var(--text-primary);
}

[data-theme="dark"] .el-tooltip__popper.is-dark {
  background-color: #4a4a4a;
  color: var(--text-primary);
}

[data-theme="dark"] .el-card {
  background-color: var(--card-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-dialog {
  background-color: var(--card-bg);
}

[data-theme="dark"] .el-dialog__header {
  border-bottom: 1px solid var(--border-primary);
}

[data-theme="dark"] .el-dialog__title {
  color: var(--text-primary);
}

[data-theme="dark"] .el-menu {
  background-color: var(--sidebar-bg);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-menu-item {
  color: var(--text-primary);
}

[data-theme="dark"] .el-menu-item:hover {
  background-color: var(--bg-hover);
}

[data-theme="dark"] .el-menu-item.is-active {
  color: var(--accent-primary);
  background-color: rgba(0, 120, 212, 0.1);
}

[data-theme="dark"] .el-sub-menu__title {
  color: var(--text-primary);
}

[data-theme="dark"] .el-table {
  background-color: var(--card-bg);
  color: var(--text-primary);
}

[data-theme="dark"] .el-table th.el-table__cell {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

[data-theme="dark"] .el-table tr {
  background-color: var(--card-bg);
}

[data-theme="dark"] .el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell {
  background-color: var(--bg-secondary);
}

[data-theme="dark"] .el-form-item__label {
  color: var(--text-secondary);
}

[data-theme="dark"] .el-textarea__inner {
  background-color: var(--input-bg);
  color: var(--text-primary);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-textarea__inner::placeholder {
  color: var(--text-placeholder);
}

[data-theme="dark"] .el-tooltip__trigger {
  color: var(--text-primary);
}

[data-theme="dark"] .el-tag {
  background-color: var(--bg-tertiary);
  border-color: var(--border-primary);
  color: var(--text-primary);
}

[data-theme="dark"] .el-divider {
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-collapse-item__header {
  background-color: var(--card-bg);
  color: var(--text-primary);
  border-color: var(--border-primary);
}

[data-theme="dark"] .el-collapse-item__content {
  background-color: var(--card-bg);
  color: var(--text-secondary);
}

[data-theme="dark"] .el-drawer {
  background-color: var(--card-bg);
}

[data-theme="dark"] .el-drawer__header {
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-primary);
}
</style>
