import io
import logging
from PyPDF2 import PdfReader
from docx import Document

logger = logging.getLogger(__name__)

class TextExtractor:
    """Extract text from various file formats"""
    
    def extract(self, file_bytes, filename):
        """
        Extract text from file based on extension
        
        Args:
            file_bytes: binary file content
            filename: original filename with extension
            
        Returns:
            str: extracted text
        """
        ext = filename.lower().split('.')[-1]
        
        if ext == 'pdf':
            return self._extract_pdf(file_bytes)
        elif ext in ['docx', 'doc']:
            return self._extract_docx(file_bytes)
        elif ext == 'txt':
            return self._extract_txt(file_bytes)
        else:
            logger.warning(f"Unsupported format: {ext}")
            return ""
    
    def _extract_pdf(self, file_bytes):
        try:
            pdf_file = io.BytesIO(file_bytes)
            reader = PdfReader(pdf_file)
            text = ""
            for page in reader.pages:
                text += page.extract_text() + "\n"
            return text.strip()
        except Exception as e:
            logger.error(f"PDF extraction failed: {e}")
            return ""
    
    def _extract_docx(self, file_bytes):
        try:
            docx_file = io.BytesIO(file_bytes)
            doc = Document(docx_file)
            text = "\n".join([para.text for para in doc.paragraphs])
            return text.strip()
        except Exception as e:
            logger.error(f"DOCX extraction failed: {e}")
            return ""
    
    def _extract_txt(self, file_bytes):
        try:
            return file_bytes.decode('utf-8').strip()
        except UnicodeDecodeError:
            try:
                return file_bytes.decode('cp1251').strip()
            except Exception as e:
                logger.error(f"TXT decoding failed: {e}")
                return ""
            