import dotenv from "dotenv";

dotenv.config();

const DEFAULT_TEMPLATE_ENGINE = "njk";

const fullURL = (path) => {
  path = path.startsWith("/") ? path : `/${path}`;

  const url =
    process.env.URL ||
    (process.env.VERCEL_PROJECT_PRODUCTION_URL
      ? `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`
      : null);

  if (!url) {
    return path;
  }

  return new URL(
    path,
    /^https?:\/\//.test(url) ? url : `https://${url}`
  ).toString();
};

/** @param {import("@11ty/eleventy").UserConfig} eleventyConfig */
export default async function (eleventyConfig) {
  eleventyConfig.setQuietMode(process.env.NODE_ENV === "production");

  eleventyConfig.addFilter("fullURL", fullURL);

  eleventyConfig.addPassthroughCopy("assets");

  return {
    dir: {
      input: "_11ty",
      output: "public",
      data: "_data",
      includes: "_includes",
      layouts: "_layouts",
      htmlTemplateEngine: DEFAULT_TEMPLATE_ENGINE,
      markdownTemplateEngine: DEFAULT_TEMPLATE_ENGINE,
    },
  };
}
