import React from "react";
import { X } from "lucide-react";

export function Button({
  children,
  className,
  variant,
  type = "button",
  ...props
}) {
  let btnClass = className || "";
  if (variant === "primary") btnClass = `primary ${btnClass}`.trim();
  else if (variant === "danger") btnClass = `danger-button ${btnClass}`.trim();
  else if (variant === "ghost") btnClass = `ghost ${btnClass}`.trim();
  else if (variant === "icon") btnClass = `icon-button ${btnClass}`.trim();

  return (
    <button type={type} className={btnClass || undefined} {...props}>
      {children}
    </button>
  );
}

export const Input = React.forwardRef(({ className, ...props }, ref) => {
  return <input ref={ref} className={className || undefined} {...props} />;
});

export const Checkbox = React.forwardRef(
  ({ label, className, wrapperClassName, ...props }, ref) => {
    if (label) {
      return (
        <label className={`checkbox-label ${wrapperClassName || ""}`.trim()}>
          <input
            ref={ref}
            type="checkbox"
            className={className || undefined}
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
        className={className || undefined}
        {...props}
      />
    );
  },
);

export const Select = React.forwardRef(
  ({ className, children, ...props }, ref) => {
    return (
      <select ref={ref} className={className || undefined} {...props}>
        {children}
      </select>
    );
  },
);

export const Textarea = React.forwardRef(({ className, ...props }, ref) => {
  return <textarea ref={ref} className={className || undefined} {...props} />;
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
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className={`modal ${className}`.trim()}
        onClick={(event) => event.stopPropagation()}
      >
        {title && (typeof title === "string" ? <h2>{title}</h2> : title)}
        {subtitle &&
          (typeof subtitle === "string" ? (
            <p className="muted">{subtitle}</p>
          ) : (
            subtitle
          ))}
        {children}
      </div>
    </div>
  );
}
