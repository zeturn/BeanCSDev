import React from "react";
import { X } from "lucide-react";
import {
  Button as WatercolorButton,
  Modal as WatercolorModal,
} from "@zeturn/watercolor-react";

function isIconElement(child) {
  if (!React.isValidElement(child)) return false;
  if (child.type === "svg") return true;
  if (typeof child.type === "string") return false;
  return Boolean(child.props?.size || child.props?.strokeWidth);
}

function trimContent(children) {
  const items = React.Children.toArray(children);
  while (items.length && typeof items[0] === "string" && !items[0].trim()) {
    items.shift();
  }
  while (
    items.length &&
    typeof items[items.length - 1] === "string" &&
    !items[items.length - 1].trim()
  ) {
    items.pop();
  }
  if (items.length === 1) return items[0];
  return items;
}

function splitButtonContent(children, startIcon, endIcon, variant) {
  const items = React.Children.toArray(children);
  if (variant === "icon" || startIcon || endIcon || items.length < 2) {
    return { content: children, startIcon, endIcon };
  }

  let contentItems = items;
  let resolvedStartIcon = startIcon;
  let resolvedEndIcon = endIcon;

  if (isIconElement(contentItems[0])) {
    resolvedStartIcon = contentItems[0];
    contentItems = contentItems.slice(1);
  }

  if (!resolvedEndIcon && contentItems.length > 1) {
    const lastItem = contentItems[contentItems.length - 1];
    if (isIconElement(lastItem)) {
      resolvedEndIcon = lastItem;
      contentItems = contentItems.slice(0, -1);
    }
  }

  return {
    content: trimContent(contentItems),
    startIcon: resolvedStartIcon,
    endIcon: resolvedEndIcon,
  };
}

export function Button({
  children,
  className,
  variant,
  type = "button",
  startIcon,
  endIcon,
  ...props
}) {
  const wcProps =
    variant === "primary"
      ? { variant: "primary", buttonStyle: "filled" }
      : variant === "danger"
        ? { variant: "error", buttonStyle: "outlined" }
        : variant === "ghost"
          ? { variant: "filled", buttonStyle: "outlined" }
          : { variant: "filled", buttonStyle: "outlined" };
  const buttonContent = splitButtonContent(children, startIcon, endIcon, variant);
  let btnClass = `beancs-button ${className || ""}`.trim();
  if (variant === "primary") btnClass = `primary ${btnClass}`.trim();
  else if (variant === "danger") btnClass = `danger-button ${btnClass}`.trim();
  else if (variant === "ghost") btnClass = `ghost ${btnClass}`.trim();
  else if (variant === "icon") btnClass = `icon-button ${btnClass}`.trim();

  return (
    <WatercolorButton
      type={type}
      size="sm"
      rounded="lg"
      className={btnClass || undefined}
      startIcon={buttonContent.startIcon}
      endIcon={buttonContent.endIcon}
      {...wcProps}
      {...props}
    >
      {buttonContent.content}
    </WatercolorButton>
  );
}

export const Input = React.forwardRef(({ className, ...props }, ref) => {
  return (
    <input
      ref={ref}
      className={`beancs-field wc-state-field ${className || ""}`.trim()}
      {...props}
    />
  );
});

export const Checkbox = React.forwardRef(
  ({ label, className, wrapperClassName, ...props }, ref) => {
    if (label) {
      return (
        <label className={`checkbox-label ${wrapperClassName || ""}`.trim()}>
          <input
            ref={ref}
            type="checkbox"
            className={`beancs-checkbox ${className || ""}`.trim()}
            {...props}
          />
          <span>{label}</span>
        </label>
      );
    }
    return (
      <input
        ref={ref}
        type="checkbox"
        className={`beancs-checkbox ${className || ""}`.trim()}
        {...props}
      />
    );
  },
);

export const Select = React.forwardRef(
  ({ className, children, ...props }, ref) => {
    return (
      <select
        ref={ref}
        className={`beancs-select wc-state-field ${className || ""}`.trim()}
        {...props}
      >
        {children}
      </select>
    );
  },
);

export const Textarea = React.forwardRef(({ className, ...props }, ref) => {
  return (
    <textarea
      ref={ref}
      className={`beancs-field beancs-textarea wc-state-field ${className || ""}`.trim()}
      {...props}
    />
  );
});

export function Drawer({
  onClose,
  title,
  subtitle,
  children,
  className = "",
  headContent,
}) {
  return (
    <div className="side-drawer-backdrop" onClick={onClose}>
      <aside
        className={`side-drawer ${className}`.trim()}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="side-drawer-head">
          <div>
            {title && (typeof title === "string" ? <h2>{title}</h2> : title)}
            {subtitle &&
              (typeof subtitle === "string" ? <p>{subtitle}</p> : subtitle)}
          </div>
          {headContent}
          <Button variant="icon" aria-label="Close" onClick={onClose}>
            <X size={16} />
          </Button>
        </div>
        {children}
      </aside>
    </div>
  );
}

export function Modal({ onClose, title, subtitle, children, className = "" }) {
  return (
    <WatercolorModal
      visible
      onClose={onClose}
      title={title}
      maxWidth={className.includes("wide-modal") ? "xl" : "md"}
      className={`beancs-modal ${className}`.trim()}
    >
      {subtitle &&
        (typeof subtitle === "string" ? (
          <p className="muted">{subtitle}</p>
        ) : (
          subtitle
        ))}
      {children}
    </WatercolorModal>
  );
}
