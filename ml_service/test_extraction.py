from text_extractor import TextExtractor

def test_txt():
    print("Testing TXT extraction...")
    extractor = TextExtractor()
    test_data = b"Hello world! This is a test document."
    result = extractor.extract(test_data, "test.txt")
    print(f"  Extracted: {len(result)} chars")
    print(f"  Content: {result}")
    assert len(result) > 0

def test_chunking():
    print("\nTesting text chunking...")
    from embedder import Embedder
    embedder = Embedder()
    
    text = "First sentence. Second sentence. Third sentence."
    chunks = embedder.chunk_text(text, chunk_size=20, overlap=5)
    print(f"  Created {len(chunks)} chunks:")
    for i, chunk in enumerate(chunks):
        print(f"    [{i}]: {chunk[:30]}...")

if __name__ == "__main__":
    print("=== Text Extraction Tests ===\n")
    test_txt()
    
    try:
        test_chunking()
    except ImportError as e:
        print(f"\nSkipping embedder test (missing deps): {e}")
    
    print("\n✓ Basic tests passed!")