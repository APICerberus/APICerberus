import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { adminApiRequest } from "@/lib/api";
import { APP_NAME, COLOR_TOKENS } from "@/lib/constants";

export type BrandingConfig = {
  app_name: string;
  logo_url: string;
  favicon_url: string;
  primary_color: string;
  accent_color: string;
  theme_mode: string;
};

const defaultBranding: BrandingConfig = {
  app_name: APP_NAME,
  logo_url: "",
  favicon_url: "",
  primary_color: COLOR_TOKENS.brand,
  accent_color: "",
  theme_mode: "system",
};

const BrandingContext = createContext<BrandingConfig>(defaultBranding);

export function useBranding() {
  return useContext(BrandingContext);
}

export function BrandingProvider({ children }: { children: ReactNode }) {
  const [branding, setBranding] = useState<BrandingConfig>(defaultBranding);

  useEffect(() => {
    adminApiRequest<BrandingConfig>("/admin/api/v1/branding/public")
      .then((data) => {
        const merged = { ...defaultBranding, ...data };
        setBranding(merged);

        // Apply branding
        if (merged.app_name) {
          document.title = merged.app_name;
        }
        if (merged.favicon_url) {
          let link = document.querySelector("link[rel='icon']") as HTMLLinkElement | null;
          if (!link) {
            link = document.createElement("link");
            link.rel = "icon";
            document.head.appendChild(link);
          }
          link.href = merged.favicon_url;
        }
        if (merged.primary_color) {
          document.documentElement.style.setProperty("--brand-primary", merged.primary_color);
        }
      })
      .catch(() => {
        // Keep defaults
      });
  }, []);

  return (
    <BrandingContext.Provider value={branding}>
      {children}
    </BrandingContext.Provider>
  );
}
