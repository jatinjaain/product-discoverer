
# Product Discoverer

Product Discoverer is a high-performance tool built in Golang for discovering product URLs from e-commerce websites. It employs two strategies: **Sitemap Parsing** and **Headless Browser Scraping** using `chromedp` for dynamic websites. The tool also supports concurrency and rotating proxies for better scalability and to avoid rate limiting.

## Getting Started

### Prerequisites

Ensure that you have the following installed on your system:

- [Go](https://golang.org/doc/install)
- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

### Environment Variables

Before running the project, set up the following environment variables in your `.env` file:

```
HEADLESS_BROWSING_WORKERS=3
INPUT_LINK_HANDLING_WORKERS=2
LINKS_LIMIT_FOR_HEADLESS_BROWSER=200
```

### Install Dependencies

Run the following command to install the necessary dependencies:

```bash
go mod vendor
```

### Build the Project

Use the following command to build the project:

```bash
go build -o product-discoverer
```

### Running the Application

After building the project, run the executable as follows:

```bash
./product-discoverer
```

### Input

You can provide multiple domains as input when running the application.

## How It Works

I am using Golang due to its superior concurrency capabilities, which enables faster execution at scale. The project implements two strategies for discovering product URLs:

### 1. Sitemap Parsing (Preferred Method)

This method fetches links from the `sitemap.xml` file, which is accessed via the `/robots.txt` file of the domain (e.g., `myntra.com/robots.txt`). Here's how the method works:

- Visit `/robots.txt` to find the link for the `sitemap.xml` file.
- If required, decompress the sitemap and traverse it to find product-related XML files.
- Once found, extract the URLs matching product links by checking predefined product URL patterns.

#### Why Sitemap Parsing First?

Sitemap parsing is **fast** and **efficient**. For example, I was able to fetch 2 million product links from Myntra within 60 seconds. This method works on around 90% of websites.

### 2. Headless Browser Scraping (Fallback Method)

If the sitemap method doesn't yield any results, we spawn multiple workers to scrape links dynamically using `chromedp`, a headless browser for handling dynamic websites.

- Scroll the webpage to a predefined limit (to avoid getting stuck in infinite scrolls) and extract product links.
- Only visit links that belong to the target domain.
- Avoid visiting the same page twice.
- The system uses rotating proxies and variable delays to avoid rate limiting by websites.

### Product Link Detection

For now, the following patterns are used to detect product links. These are taken from below popular e-commerce websites:

| Domain           | Product Link Substrings                                   |
|------------------|-----------------------------------------------------------|
| Souled Store      | `/product/`                                               |
| Littlebox         | `/product/`                                               |
| Newme             | `/product/`                                               |
| Myntra            | `/buy`                                                    |
| AJIO              | `/p/`                                                     |
| Flipkart          | `/p/`                                                     |
| Nike              | `/t/`                                                     |
| Snitch            | `/products/`                                              |
| Libas             | `/products/`                                              |
| Uniqlo            | `/products/`                                              |
| Comet             | `/products/`                                              |
| Urban Monkey      | `/products/`                                              |
| Virgio            | `/products/`                                              |
| Rareism           | `/products/`                                              |
| H&M               | `/productpage.0970818065.html`                            |
| FirstCry          | `/product-detail`                                         |
| Amazon            | `/dp/`                                                    |

### Rotating Proxies and Variable Delay

To avoid rate-limiting and IP bans, the tool supports rotating proxies and adds variable delays between requests. You will need to pass paid proxy URLs for this feature to be effective.

## Remaining Improvements

- **Product Substring Detection:** We could find product-specific substrings dynamically by visiting pages and looking for common texts like "Add to Cart", "Buy Now", "Product Details",  "product information", "reviews", "add to wish list" etc.
And we find 10 such links which have above texts written in it and we will find common substring except domain in those links.
- **Concurrency for Sitemap Parsing:** Currently, the sitemap parsing is single-threaded. While it is already very fast, introducing concurrency could make it even faster for larger websites.
- **Output Storage:** Currently, results are written to a `.txt` file named after the domain. Further improvements can include structured output formats like JSON or CSV.

## Example Usage

1. Fetching links using `sitemap.xml` is **very fast** and works on 90% of websites. For `myntra.com`, I fetched 2 million product links in just **60 seconds**.
2. If the `sitemap.xml` method doesn't work, the headless browser method scrapes and scrolls dynamic websites like Myntra and Amazon.

---

## License

This project is licensed under the Creative Commons Attribution-NonCommercial 4.0 International (CC BY-NC 4.0). See the [LICENSE](LICENSE) file for details.

---

### Future Improvements

- Implement dynamic product substring detection.
- Concurrency for sitemap parsing for even faster processing on large sites.

